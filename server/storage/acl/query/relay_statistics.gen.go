// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"

	"gorm.io/plugin/dbresolver"

	"github.com/doraemonkeys/WindSend-Relay/storage/acl/model"
)

func newRelayStatistic(db *gorm.DB, opts ...gen.DOOption) relayStatistic {
	_relayStatistic := relayStatistic{}

	_relayStatistic.relayStatisticDo.UseDB(db, opts...)
	_relayStatistic.relayStatisticDo.UseModel(&model.RelayStatistic{})

	tableName := _relayStatistic.relayStatisticDo.TableName()
	_relayStatistic.ALL = field.NewAsterisk(tableName)
	_relayStatistic.ID = field.NewString(tableName, "id")
	_relayStatistic.CreatedAt = field.NewTime(tableName, "created_at")
	_relayStatistic.UpdatedAt = field.NewTime(tableName, "updated_at")
	_relayStatistic.CustomName = field.NewString(tableName, "custom_name")
	_relayStatistic.TotalRelayCount = field.NewInt(tableName, "total_relay_count")
	_relayStatistic.TotalRelayErrCount = field.NewInt(tableName, "total_relay_err_count")
	_relayStatistic.TotalRelayOfflineCount = field.NewInt(tableName, "total_relay_offline_count")
	_relayStatistic.TotalRelayMs = field.NewInt64(tableName, "total_relay_ms")
	_relayStatistic.TotalRelayBytes = field.NewInt64(tableName, "total_relay_bytes")

	_relayStatistic.fillFieldMap()

	return _relayStatistic
}

type relayStatistic struct {
	relayStatisticDo

	ALL                    field.Asterisk
	ID                     field.String
	CreatedAt              field.Time
	UpdatedAt              field.Time
	CustomName             field.String
	TotalRelayCount        field.Int
	TotalRelayErrCount     field.Int
	TotalRelayOfflineCount field.Int
	TotalRelayMs           field.Int64
	TotalRelayBytes        field.Int64

	fieldMap map[string]field.Expr
}

func (r relayStatistic) Table(newTableName string) *relayStatistic {
	r.relayStatisticDo.UseTable(newTableName)
	return r.updateTableName(newTableName)
}

func (r relayStatistic) As(alias string) *relayStatistic {
	r.relayStatisticDo.DO = *(r.relayStatisticDo.As(alias).(*gen.DO))
	return r.updateTableName(alias)
}

func (r *relayStatistic) updateTableName(table string) *relayStatistic {
	r.ALL = field.NewAsterisk(table)
	r.ID = field.NewString(table, "id")
	r.CreatedAt = field.NewTime(table, "created_at")
	r.UpdatedAt = field.NewTime(table, "updated_at")
	r.CustomName = field.NewString(table, "custom_name")
	r.TotalRelayCount = field.NewInt(table, "total_relay_count")
	r.TotalRelayErrCount = field.NewInt(table, "total_relay_err_count")
	r.TotalRelayOfflineCount = field.NewInt(table, "total_relay_offline_count")
	r.TotalRelayMs = field.NewInt64(table, "total_relay_ms")
	r.TotalRelayBytes = field.NewInt64(table, "total_relay_bytes")

	r.fillFieldMap()

	return r
}

func (r *relayStatistic) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := r.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (r *relayStatistic) fillFieldMap() {
	r.fieldMap = make(map[string]field.Expr, 9)
	r.fieldMap["id"] = r.ID
	r.fieldMap["created_at"] = r.CreatedAt
	r.fieldMap["updated_at"] = r.UpdatedAt
	r.fieldMap["custom_name"] = r.CustomName
	r.fieldMap["total_relay_count"] = r.TotalRelayCount
	r.fieldMap["total_relay_err_count"] = r.TotalRelayErrCount
	r.fieldMap["total_relay_offline_count"] = r.TotalRelayOfflineCount
	r.fieldMap["total_relay_ms"] = r.TotalRelayMs
	r.fieldMap["total_relay_bytes"] = r.TotalRelayBytes
}

func (r relayStatistic) clone(db *gorm.DB) relayStatistic {
	r.relayStatisticDo.ReplaceConnPool(db.Statement.ConnPool)
	return r
}

func (r relayStatistic) replaceDB(db *gorm.DB) relayStatistic {
	r.relayStatisticDo.ReplaceDB(db)
	return r
}

type relayStatisticDo struct{ gen.DO }

type IRelayStatisticDo interface {
	gen.SubQuery
	Debug() IRelayStatisticDo
	WithContext(ctx context.Context) IRelayStatisticDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IRelayStatisticDo
	WriteDB() IRelayStatisticDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IRelayStatisticDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IRelayStatisticDo
	Not(conds ...gen.Condition) IRelayStatisticDo
	Or(conds ...gen.Condition) IRelayStatisticDo
	Select(conds ...field.Expr) IRelayStatisticDo
	Where(conds ...gen.Condition) IRelayStatisticDo
	Order(conds ...field.Expr) IRelayStatisticDo
	Distinct(cols ...field.Expr) IRelayStatisticDo
	Omit(cols ...field.Expr) IRelayStatisticDo
	Join(table schema.Tabler, on ...field.Expr) IRelayStatisticDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IRelayStatisticDo
	RightJoin(table schema.Tabler, on ...field.Expr) IRelayStatisticDo
	Group(cols ...field.Expr) IRelayStatisticDo
	Having(conds ...gen.Condition) IRelayStatisticDo
	Limit(limit int) IRelayStatisticDo
	Offset(offset int) IRelayStatisticDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IRelayStatisticDo
	Unscoped() IRelayStatisticDo
	Create(values ...*model.RelayStatistic) error
	CreateInBatches(values []*model.RelayStatistic, batchSize int) error
	Save(values ...*model.RelayStatistic) error
	First() (*model.RelayStatistic, error)
	Take() (*model.RelayStatistic, error)
	Last() (*model.RelayStatistic, error)
	Find() ([]*model.RelayStatistic, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.RelayStatistic, err error)
	FindInBatches(result *[]*model.RelayStatistic, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.RelayStatistic) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IRelayStatisticDo
	Assign(attrs ...field.AssignExpr) IRelayStatisticDo
	Joins(fields ...field.RelationField) IRelayStatisticDo
	Preload(fields ...field.RelationField) IRelayStatisticDo
	FirstOrInit() (*model.RelayStatistic, error)
	FirstOrCreate() (*model.RelayStatistic, error)
	FindByPage(offset int, limit int) (result []*model.RelayStatistic, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Rows() (*sql.Rows, error)
	Row() *sql.Row
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IRelayStatisticDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (r relayStatisticDo) Debug() IRelayStatisticDo {
	return r.withDO(r.DO.Debug())
}

func (r relayStatisticDo) WithContext(ctx context.Context) IRelayStatisticDo {
	return r.withDO(r.DO.WithContext(ctx))
}

func (r relayStatisticDo) ReadDB() IRelayStatisticDo {
	return r.Clauses(dbresolver.Read)
}

func (r relayStatisticDo) WriteDB() IRelayStatisticDo {
	return r.Clauses(dbresolver.Write)
}

func (r relayStatisticDo) Session(config *gorm.Session) IRelayStatisticDo {
	return r.withDO(r.DO.Session(config))
}

func (r relayStatisticDo) Clauses(conds ...clause.Expression) IRelayStatisticDo {
	return r.withDO(r.DO.Clauses(conds...))
}

func (r relayStatisticDo) Returning(value interface{}, columns ...string) IRelayStatisticDo {
	return r.withDO(r.DO.Returning(value, columns...))
}

func (r relayStatisticDo) Not(conds ...gen.Condition) IRelayStatisticDo {
	return r.withDO(r.DO.Not(conds...))
}

func (r relayStatisticDo) Or(conds ...gen.Condition) IRelayStatisticDo {
	return r.withDO(r.DO.Or(conds...))
}

func (r relayStatisticDo) Select(conds ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Select(conds...))
}

func (r relayStatisticDo) Where(conds ...gen.Condition) IRelayStatisticDo {
	return r.withDO(r.DO.Where(conds...))
}

func (r relayStatisticDo) Order(conds ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Order(conds...))
}

func (r relayStatisticDo) Distinct(cols ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Distinct(cols...))
}

func (r relayStatisticDo) Omit(cols ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Omit(cols...))
}

func (r relayStatisticDo) Join(table schema.Tabler, on ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Join(table, on...))
}

func (r relayStatisticDo) LeftJoin(table schema.Tabler, on ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.LeftJoin(table, on...))
}

func (r relayStatisticDo) RightJoin(table schema.Tabler, on ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.RightJoin(table, on...))
}

func (r relayStatisticDo) Group(cols ...field.Expr) IRelayStatisticDo {
	return r.withDO(r.DO.Group(cols...))
}

func (r relayStatisticDo) Having(conds ...gen.Condition) IRelayStatisticDo {
	return r.withDO(r.DO.Having(conds...))
}

func (r relayStatisticDo) Limit(limit int) IRelayStatisticDo {
	return r.withDO(r.DO.Limit(limit))
}

func (r relayStatisticDo) Offset(offset int) IRelayStatisticDo {
	return r.withDO(r.DO.Offset(offset))
}

func (r relayStatisticDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IRelayStatisticDo {
	return r.withDO(r.DO.Scopes(funcs...))
}

func (r relayStatisticDo) Unscoped() IRelayStatisticDo {
	return r.withDO(r.DO.Unscoped())
}

func (r relayStatisticDo) Create(values ...*model.RelayStatistic) error {
	if len(values) == 0 {
		return nil
	}
	return r.DO.Create(values)
}

func (r relayStatisticDo) CreateInBatches(values []*model.RelayStatistic, batchSize int) error {
	return r.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (r relayStatisticDo) Save(values ...*model.RelayStatistic) error {
	if len(values) == 0 {
		return nil
	}
	return r.DO.Save(values)
}

func (r relayStatisticDo) First() (*model.RelayStatistic, error) {
	if result, err := r.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.RelayStatistic), nil
	}
}

func (r relayStatisticDo) Take() (*model.RelayStatistic, error) {
	if result, err := r.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.RelayStatistic), nil
	}
}

func (r relayStatisticDo) Last() (*model.RelayStatistic, error) {
	if result, err := r.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.RelayStatistic), nil
	}
}

func (r relayStatisticDo) Find() ([]*model.RelayStatistic, error) {
	result, err := r.DO.Find()
	return result.([]*model.RelayStatistic), err
}

func (r relayStatisticDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.RelayStatistic, err error) {
	buf := make([]*model.RelayStatistic, 0, batchSize)
	err = r.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (r relayStatisticDo) FindInBatches(result *[]*model.RelayStatistic, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return r.DO.FindInBatches(result, batchSize, fc)
}

func (r relayStatisticDo) Attrs(attrs ...field.AssignExpr) IRelayStatisticDo {
	return r.withDO(r.DO.Attrs(attrs...))
}

func (r relayStatisticDo) Assign(attrs ...field.AssignExpr) IRelayStatisticDo {
	return r.withDO(r.DO.Assign(attrs...))
}

func (r relayStatisticDo) Joins(fields ...field.RelationField) IRelayStatisticDo {
	for _, _f := range fields {
		r = *r.withDO(r.DO.Joins(_f))
	}
	return &r
}

func (r relayStatisticDo) Preload(fields ...field.RelationField) IRelayStatisticDo {
	for _, _f := range fields {
		r = *r.withDO(r.DO.Preload(_f))
	}
	return &r
}

func (r relayStatisticDo) FirstOrInit() (*model.RelayStatistic, error) {
	if result, err := r.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.RelayStatistic), nil
	}
}

func (r relayStatisticDo) FirstOrCreate() (*model.RelayStatistic, error) {
	if result, err := r.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.RelayStatistic), nil
	}
}

func (r relayStatisticDo) FindByPage(offset int, limit int) (result []*model.RelayStatistic, count int64, err error) {
	result, err = r.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = r.Offset(-1).Limit(-1).Count()
	return
}

func (r relayStatisticDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = r.Count()
	if err != nil {
		return
	}

	err = r.Offset(offset).Limit(limit).Scan(result)
	return
}

func (r relayStatisticDo) Scan(result interface{}) (err error) {
	return r.DO.Scan(result)
}

func (r relayStatisticDo) Delete(models ...*model.RelayStatistic) (result gen.ResultInfo, err error) {
	return r.DO.Delete(models)
}

func (r *relayStatisticDo) withDO(do gen.Dao) *relayStatisticDo {
	r.DO = *do.(*gen.DO)
	return r
}
