package admin

import (
	"crypto/rand"
	"net/http"
	"strings"
	"time"

	"github.com/doraemonkeys/WindSend-Relay/admin/dto"
	"github.com/doraemonkeys/WindSend-Relay/config"
	"github.com/doraemonkeys/WindSend-Relay/relay"
	"github.com/doraemonkeys/WindSend-Relay/storage"
	"github.com/doraemonkeys/WindSend-Relay/tool"
	"github.com/doraemonkeys/doraemon"
	"github.com/doraemonkeys/doraemon/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminServer struct {
	cfg     *config.AdminConfig
	storage storage.Storage
	relay   *relay.Relay
	router  *gin.Engine
	j       *jwt.JWT[string]
}

func NewAdminServer(relay *relay.Relay, storage storage.Storage, cfg *config.AdminConfig) *AdminServer {
	salt, err := storage.GetAdminSalt()
	if err != nil {
		zap.L().Fatal("failed to get admin salt", zap.Error(err))
	}
	if salt == nil {
		salt = make([]byte, 16)
		_, err := rand.Read(salt)
		if err != nil {
			zap.L().Fatal("failed to generate salt", zap.Error(err))
		}
		err = storage.SetAdminSalt(salt)
		if err != nil {
			zap.L().Fatal("failed to set admin salt", zap.Error(err))
		}
	}
	ph := tool.AES192KeyKDF(cfg.Password, salt)
	j, err := jwt.NewHS256JWT[string](ph)
	if err != nil {
		zap.L().Fatal("failed to create jwt", zap.Error(err))
	}
	return &AdminServer{relay: relay, storage: storage, cfg: cfg, j: j}
}

func (s *AdminServer) SetupRouter() {
	// gin.SetMode(gin.ReleaseMode)
	s.router = gin.Default()
	s.router.SetTrustedProxies([]string{"127.0.0.1", "::1", "localhost"})

	// TODO: Remove this
	// s.router.Use(cors.New(cors.Config{
	// 	AllowAllOrigins:  true,
	// 	AllowCredentials: true,
	// 	AllowHeaders:     []string{"*"},
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	ExposeHeaders:    []string{"Content-Length"},
	// 	// AllowOrigins:     []string{"*"},
	// }))

	s.router.NoRoute(s.handleNoRoute)
	s.router.Static("/home", config.WebStaticDir)
	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/home/index.html")
	})

	api := s.router.Group("/api")
	{
		api.POST("/login", s.handleLogin)
		api.GET("/conn/statistic", s.authMiddleware(), s.handleGetConnectionStatistic)
		api.GET("/conn/status", s.authMiddleware(), s.handleGetConnectionStatus)
		api.GET("/conn/close/:id", s.authMiddleware(), s.handleCloseConnection)
	}
}

func (s *AdminServer) Run() {
	s.SetupRouter()
	err := s.router.Run(s.cfg.Addr)
	if err != nil {
		zap.L().Fatal("failed to run admin server", zap.Error(err))
	}
}

func (s *AdminServer) handleNoRoute(c *gin.Context) {
	zap.L().Info("no route found", zap.String("path", c.Request.URL.Path))
	c.JSON(http.StatusNotFound, gin.H{
		"message": "not found",
	})
}

func (s *AdminServer) handleLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	ph, err := doraemon.ComputeSHA256Hex(strings.NewReader(s.cfg.Password))
	if err != nil {
		zap.L().Error("failed to compute sha256", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to compute sha256",
		})
		return
	}
	if username != s.cfg.User || !strings.EqualFold(password, ph) {
		zap.L().Error("invalid username or password", zap.String("username", username), zap.String("password", password))
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
		return
	}
	token, err := s.j.CreateDefaultToken(username, time.Now().Add(time.Hour*24*30))
	if err != nil {
		zap.L().Error("failed to create jwt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to create jwt",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

func (s *AdminServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			c.Abort()
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")
		claims, err := s.j.ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			c.Abort()
			return
		}
		err = s.j.VerifyTokenOnlySignInfo(token, claims.SignInfo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			c.Abort()
			return
		}
		c.Set("username", claims.SignInfo)
		c.Next()
	}
}

func (s *AdminServer) handleCloseConnection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "id is required",
		})
		return
	}
	s.relay.RemoveLongConnection(id)
	c.Status(http.StatusOK)
}

func (s *AdminServer) handleGetConnectionStatus(c *gin.Context) {
	var resp = make([]dto.ActiveConnection, 0)

	alives := s.relay.GetAllStatus()
	for _, alive := range alives {
		stat, err := s.storage.GetHistoryStatisticByID(alive.ID)
		if err != nil {
			zap.L().Error("failed to get history statistic", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to get history statistic",
			})
			return
		}
		resp = append(resp, dto.ActiveConnection{
			ID:          alive.ID,
			ReqAddr:     alive.ReqAddr,
			ConnectTime: alive.ConnectTime,
			LastActive:  alive.LastActive,
			Relaying:    alive.Relaying,
			History: dto.HistoryStatistic{
				ID:                     stat.ID,
				CreatedAt:              stat.CreatedAt,
				UpdatedAt:              stat.UpdatedAt,
				TotalRelayCount:        stat.TotalRelayCount,
				TotalRelayErrCount:     stat.TotalRelayErrCount,
				TotalRelayOfflineCount: stat.TotalRelayOfflineCount,
				TotalRelayMs:           stat.TotalRelayMs,
				TotalRelayBytes:        stat.TotalRelayBytes,
			},
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (s *AdminServer) handleGetConnectionStatistic(c *gin.Context) {
	req := dto.ReqHistoryStatistic{}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request",
		})
		return
	}
	stats, total, err := s.storage.GetHistoryStatistic(req.Page, req.PageSize, req.SortBy, req.SortType)
	if err != nil {
		zap.L().Error("failed to get history statistic", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get history statistic",
		})
		return
	}
	resp := dto.RespHistoryStatistic{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	var list = make([]dto.HistoryStatistic, 0)
	for _, stat := range stats {
		list = append(list, dto.HistoryStatistic{
			ID:                     stat.ID,
			CreatedAt:              stat.CreatedAt,
			UpdatedAt:              stat.UpdatedAt,
			TotalRelayCount:        stat.TotalRelayCount,
			TotalRelayErrCount:     stat.TotalRelayErrCount,
			TotalRelayOfflineCount: stat.TotalRelayOfflineCount,
			TotalRelayMs:           stat.TotalRelayMs,
			TotalRelayBytes:        stat.TotalRelayBytes,
		})
	}
	resp.List = list
	c.JSON(http.StatusOK, resp)
}
