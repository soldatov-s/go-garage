package miniorm

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/x/stringsx"
)

// Predefined fields
const (
	defaultIDField        = "id"
	defaultCreatedAtField = "created_at"
	defaultUpdatedAtField = "updated_at"
	defaultDeletedAtField = "deleted_at"
)

var ErrNotContainPredefinedField = errors.New("not contain predefined field")

type PredefinedFields struct {
	IDFieldName    string
	CreatedAtField string
	UpdatedAtField string
	DeletedAtField string
}

type PredefinedFieldsOption func(*PredefinedFields)

func WithCustomIDField(name string) PredefinedFieldsOption {
	return func(c *PredefinedFields) {
		c.IDFieldName = name
	}
}

func WithCustomCreatedAtField(name string) PredefinedFieldsOption {
	return func(c *PredefinedFields) {
		c.CreatedAtField = name
	}
}

func WithCustomUpdatedAtField(name string) PredefinedFieldsOption {
	return func(c *PredefinedFields) {
		c.UpdatedAtField = name
	}
}

func WithCustomDeletedAtField(name string) PredefinedFieldsOption {
	return func(c *PredefinedFields) {
		c.DeletedAtField = name
	}
}

func NewPredefinedFields(opts ...PredefinedFieldsOption) *PredefinedFields {
	predefinedFields := &PredefinedFields{
		IDFieldName:    defaultIDField,
		CreatedAtField: defaultCreatedAtField,
		UpdatedAtField: defaultUpdatedAtField,
		DeletedAtField: defaultDeletedAtField,
	}

	for _, opt := range opts {
		opt(predefinedFields)
	}
	return predefinedFields
}

func (p *PredefinedFields) GetPredefinedFields() []string {
	return []string{
		p.IDFieldName,
		p.CreatedAtField,
		p.UpdatedAtField,
		p.DeletedAtField,
	}
}

type ORMDeps struct {
	Conn             **sqlx.DB
	Target           string
	Data             interface{}
	PredefinedFields *PredefinedFields
}

type ORM struct {
	requestParams    []string
	conn             **sqlx.DB
	target           string
	stmtCache        *sqlx.NamedStmt
	predefinedFields *PredefinedFields
}

func NewORM(deps *ORMDeps) (*ORM, error) {
	requestParam := buildRequestParam(deps.Data)

	predefinedFields := deps.PredefinedFields.GetPredefinedFields()
	for _, predefinedField := range predefinedFields {
		isContain := false
		for _, v := range requestParam {
			if v == predefinedField {
				isContain = true
				break
			}
		}
		if !isContain {
			return nil, errors.Wrapf(ErrNotContainPredefinedField, "%s", predefinedField)
		}
	}

	miniORM := &ORM{
		conn:             deps.Conn,
		target:           deps.Target,
		requestParams:    requestParam,
		predefinedFields: deps.PredefinedFields,
	}

	return miniORM, nil
}

func buildRequestParam(data interface{}) []string {
	Values := reflect.ValueOf(data).Elem()
	res := make([]string, 0, Values.NumField())
	requestParams(&res, valueFromPtr(data))
	return res
}

func valueFromPtr(s interface{}) interface{} {
	if s == nil {
		return nil
	}

	if reflect.TypeOf(s).Kind() == reflect.Ptr {
		if !reflect.ValueOf(s).IsNil() {
			return valueFromPtr(reflect.ValueOf(s).Elem().Interface())
		}
		return valueFromPtr(reflect.New(reflect.TypeOf(s).Elem()).Interface())
	}

	return s
}

func requestParams(res *[]string, data interface{}) {
	Types := reflect.TypeOf(data)
	Values := reflect.ValueOf(data)

	for i := 0; i < Types.NumField(); i++ {
		if Types.Field(i).Anonymous {
			requestParams(res, Values.Field(i).Interface())
		}
		if tag := Types.Field(i).Tag.Get("db"); tag != "" {
			*res = append(*res, tag)
		}
	}
}

// SelectByID select data from target by id
func (h *ORM) SelectByID(ctx context.Context, id int64, dest interface{}) error {
	conn := *h.conn
	sqlRequest := conn.Rebind(stringsx.JoinStrings(" ", "SELECT * FROM", h.target, "WHERE "+h.predefinedFields.IDFieldName+"=$1"))
	if err := conn.GetContext(ctx, dest, sqlRequest, id); err != nil {
		return errors.Wrap(err, "get data")
	}

	return nil
}

// HardDeleteByID hard deletes data from target by id
func (h *ORM) HardDeleteByID(ctx context.Context, id int64) error {
	conn := *h.conn
	sqlRequest := conn.Rebind(stringsx.JoinStrings(" ", "DELETE FROM", h.target, "WHERE "+h.predefinedFields.IDFieldName+"=$1"))
	if _, err := conn.ExecContext(ctx, sqlRequest, id); err != nil {
		return errors.Wrap(err, "hard delete data")
	}

	return nil
}

// SoftDeleteByID soft deletes data from target by id
// It marks data as deleted by changing deleted_at field
func (h *ORM) SoftDeleteByID(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	conn := *h.conn
	sqlRequest := conn.Rebind(
		stringsx.JoinStrings(" ", "UPDATE", h.target, "SET",
			h.predefinedFields.UpdatedAtField+"=$1",
			h.predefinedFields.DeletedAtField+"=$2",
			"WHERE "+h.predefinedFields.IDFieldName+"=$3"))
	if _, err := conn.ExecContext(ctx, sqlRequest, now, now, id); err != nil {
		return errors.Wrap(err, "soft delete data")
	}

	return nil
}

func (h *ORM) getCachedStmt(ctx context.Context) (*sqlx.NamedStmt, error) {
	if h.stmtCache != nil {
		return h.stmtCache, nil
	}

	var err error
	conn := *h.conn
	sqlRequest := conn.Rebind(
		stringsx.JoinStrings(" ", "INSERT INTO", h.target, "("+strings.Join(h.requestParams, ", ")+")",
			"VALUES", "("+":"+strings.Join(h.requestParams, ", :")+") RETURNING *"))

	h.stmtCache, err = conn.PrepareNamedContext(ctx, sqlRequest)
	if err != nil {
		return nil, errors.Wrap(err, "prepeare named")
	}

	return h.stmtCache, nil
}

// InsertInto inserts new data to target
func (h *ORM) InsertInto(ctx context.Context, src, dest interface{}) error {
	stmt, err := h.getCachedStmt(ctx)
	if err != nil {
		return errors.Wrap(err, "get cached stmt")
	}

	if err = stmt.Get(dest, src); err != nil {
		return errors.Wrap(err, "insert data")
	}

	return nil
}

func (h *ORM) Close() error {
	if err := h.stmtCache.Close(); err != nil {
		return errors.Wrap(err, "close stmt")
	}
	return nil
}

// Update data in target by id
// ID taken from passed data
func (h *ORM) Update(ctx context.Context, target string, data interface{}) (interface{}, error) {
	conn := *h.conn
	query := h.BuildQuery()

	sqlRequest := conn.Rebind(
		stringsx.JoinStrings(" ", "UPDATE "+target+" SET ", strings.Join(query, ", "),
			"WHERE "+h.predefinedFields.IDFieldName+"=:"+h.predefinedFields.IDFieldName))
	if _, err := conn.NamedExecContext(ctx, sqlRequest, data); err != nil {
		return nil, errors.Wrap(err, "update data")
	}

	return data, nil
}

// RequestParams returns sql request parameters for data
func (h *ORM) RequestParams() []string {
	return h.requestParams
}

// RequestParamsWithout returns sql request parameters for data without
// removed fields
func (h *ORM) RequestParamsWithout(remove ...string) []string {
	sqlParams := h.RequestParams()
	res := make([]string, 0, len(sqlParams))
	for _, v := range sqlParams {
		find := false
		for _, vv := range remove {
			if v == vv {
				find = true
				break
			}
		}
		if find {
			continue
		}
		res = append(res, v)
	}
	return res
}

func (h *ORM) BuildQuery() []string {
	sqlPrams := h.RequestParams()
	query := make([]string, 0, len(sqlPrams))
	for _, param := range sqlPrams {
		query = append(query, param+"=:"+param)
	}
	return query
}
