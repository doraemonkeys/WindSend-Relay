package storage

import (
	"encoding/base64"
	"errors"

	"github.com/doraemonkeys/WindSend-Relay/pkg"
	"github.com/doraemonkeys/WindSend-Relay/storage/acl/model"
	"github.com/doraemonkeys/WindSend-Relay/storage/acl/query"
	"go.uber.org/zap"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const (
	RelayStatisticBucket = "relay_statistic"
)

type Storage struct {
	db *gorm.DB
}

func NewStorage(path string) Storage {
	db := pkg.NewSqliteDB(path, zap.L())
	err := db.AutoMigrate(&model.RelayStatistic{})
	if err != nil {
		panic(err)
	}
	return Storage{
		db: db,
	}
}

func (s Storage) GetKeyValue(key string) (string, error) {
	q := query.Use(s.db)
	kv, err := q.KeyValue.Where(q.KeyValue.Key.Eq(key)).First()
	if err != nil {
		return "", err
	}
	return kv.Value, nil
}

func (s Storage) SetKeyValue(key string, value string) error {
	q := query.Use(s.db)
	err := q.Transaction(func(tx *query.Query) error {
		m, err := tx.KeyValue.Where(tx.KeyValue.Key.Eq(key)).FirstOrCreate()
		if err != nil {
			return err
		}
		m.Value = value
		return tx.KeyValue.Save(m)
	})
	return err
}

func (s Storage) GetAdminSalt() ([]byte, error) {
	salt, err := s.GetKeyValue("admin_salt")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return nil, err
	}
	return saltBytes, nil
}

func (s Storage) SetAdminSalt(salt []byte) error {
	return s.SetKeyValue("admin_salt", base64.StdEncoding.EncodeToString(salt))
}

func (s Storage) GetRelayStatistic(id string) (*model.RelayStatistic, error) {
	q := query.Use(s.db)
	stat, err := q.RelayStatistic.Where(q.RelayStatistic.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return stat, nil
}

func (s Storage) AddRelayStatistic(id string, success bool, ms int, bytes int64) {
	q := query.Use(s.db)
	q.Transaction(func(tx *query.Query) error {
		stat, err := tx.RelayStatistic.Where(tx.RelayStatistic.ID.Eq(id)).FirstOrCreate()
		if err != nil {
			zap.L().Error("add relay statistic failed", zap.Error(err))
			return err
		}
		stat.TotalRelayCount++
		stat.TotalRelayMs += int64(ms)
		stat.TotalRelayBytes += bytes
		if !success {
			stat.TotalRelayErrCount++
		}
		r, err := tx.RelayStatistic.Updates(stat)
		if err != nil {
			zap.L().Error("save relay statistic failed", zap.Error(err))
			return err
		}
		if r.RowsAffected == 0 {
			zap.L().Error("unexpected: relay statistic not found", zap.String("id", id))
		}
		return nil
	})
}

func (s Storage) IncrementRelayOfflineCount(id string) {
	q := query.Use(s.db)
	q.Transaction(func(tx *query.Query) error {
		_, err := tx.RelayStatistic.Where(tx.RelayStatistic.ID.Eq(id)).FirstOrCreate()
		if err != nil {
			zap.L().Error("increment relay offline count failed", zap.Error(err))
			return err
		}
		r, err := tx.RelayStatistic.Where(tx.RelayStatistic.ID.Eq(id)).UpdateSimple(tx.RelayStatistic.TotalRelayOfflineCount.Add(1))
		if err != nil {
			zap.L().Error("save relay statistic failed", zap.Error(err))
			return err
		}
		if r.RowsAffected == 0 {
			zap.L().Error("unexpected: relay statistic not found", zap.String("id", id))
		}
		return nil
	})
}

func (s Storage) GetHistoryStatisticByID(id string) (*model.RelayStatistic, error) {
	q := query.Use(s.db)
	stat, err := q.RelayStatistic.Where(q.RelayStatistic.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return stat, nil
}

func (s Storage) GetHistoryStatistic(page, pageSize int, sortBy string, sortType string) ([]*model.RelayStatistic, int64, error) {
	if sortBy == "" {
		return s.getHistoryStatistic(page, pageSize)
	}
	return s.getHistoryStatisticBySort(page, pageSize, sortBy, sortType)
}

func (s Storage) getHistoryStatistic(page, pageSize int) ([]*model.RelayStatistic, int64, error) {
	q := query.Use(s.db)
	stats, count, err := q.RelayStatistic.FindByPage(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return stats, count, nil
}

func (s Storage) getHistoryStatisticBySort(page, pageSize int, sortBy string, sortType string) ([]*model.RelayStatistic, int64, error) {
	q := query.Use(s.db)
	if !s.checkSortType(sortType) {
		return nil, 0, errors.New("invalid sort type")
	}
	if !s.checkSortBy(sortBy) {
		return nil, 0, errors.New("invalid sort by")
	}
	// q.RelayStatistic.TotalRelayBytes
	desc := sortType == "desc"
	var orderExpr field.Expr
	switch sortBy {
	case q.RelayStatistic.TotalRelayCount.ColumnName().String():
		if desc {
			orderExpr = q.RelayStatistic.TotalRelayCount.Desc()
		} else {
			orderExpr = q.RelayStatistic.TotalRelayCount.Asc()
		}
	case q.RelayStatistic.TotalRelayMs.ColumnName().String():
		if desc {
			orderExpr = q.RelayStatistic.TotalRelayMs.Desc()
		} else {
			orderExpr = q.RelayStatistic.TotalRelayMs.Asc()
		}
	case q.RelayStatistic.TotalRelayBytes.ColumnName().String():
		if desc {
			orderExpr = q.RelayStatistic.TotalRelayBytes.Desc()
		} else {
			orderExpr = q.RelayStatistic.TotalRelayBytes.Asc()
		}
	}
	stats, err := q.RelayStatistic.Offset(page).Limit(pageSize).Order(orderExpr).Find()
	if err != nil {
		return nil, 0, err
	}
	count, err := q.RelayStatistic.Offset(-1).Limit(-1).Count()
	if err != nil {
		return nil, 0, err
	}
	return stats, count, nil
}

func (s Storage) checkSortType(sortType string) bool {
	return sortType == "asc" || sortType == "desc"
}

func (s Storage) checkSortBy(sortBy string) bool {
	return sortBy == query.RelayStatistic.TotalRelayCount.ColumnName().String() ||
		sortBy == query.RelayStatistic.TotalRelayMs.ColumnName().String() ||
		sortBy == query.RelayStatistic.TotalRelayBytes.ColumnName().String()
}
