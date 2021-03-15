package sql

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	gogarage "github.com/soldatov-s/go-garage"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/utils"
)

const (
	HelperItem gogarage.GarageItem = "sqlhelper"
)

// RequestParameter return SQL params for struct
type RequestParameter interface {
	SQLParamsRequest(ctx context.Context) []string
}

// SelectByID select data from target by id
func SelectByID(conn *sqlx.DB, target string, id int64, data interface{}) error {
	if conn == nil {
		return db.ErrDBConnNotEstablished
	}

	err := conn.Get(data, conn.Rebind(utils.JoinStrings(" ", "SELECT * FROM", target, "WHERE id=$1")), id)
	if err != nil {
		return err
	}

	return nil
}

// HardDeleteByID hard deletes data from target by id
func HardDeleteByID(conn *sqlx.DB, target string, id int64) (err error) {
	if conn == nil {
		return db.ErrDBConnNotEstablished
	}

	_, err = conn.Exec(conn.Rebind(utils.JoinStrings(" ", "DELETE FROM", target, "WHERE id=$1")), id)
	if err != nil {
		return err
	}

	return nil
}

// SoftDeleteByID soft deletes data from target by id
// It marks data as deleted by changing deleted_at field
func SoftDeleteByID(conn *sqlx.DB, target string, id int64) (err error) {
	if conn == nil {
		return db.ErrDBConnNotEstablished
	}

	now := time.Now().UTC()
	_, err = conn.Exec(
		conn.Rebind(utils.JoinStrings(" ", "UPDATE", target, "SET", "updated_at=$1, deleted_at=$2", "WHERE id=$3")), now, now, id,
	)
	if err != nil {
		return err
	}

	return nil
}

// InsertInto inserts new data to target
func InsertInto(ctx context.Context, conn *sqlx.DB, target string, data RequestParameter) (interface{}, error) {
	if conn == nil {
		return nil, db.ErrDBConnNotEstablished
	}

	stmt, err := conn.PrepareNamed(
		conn.Rebind(utils.JoinStrings(" ", "INSERT INTO", target, "("+strings.Join(data.SQLParamsRequest(ctx), ", ")+")",
			"VALUES", "("+":"+strings.Join(data.SQLParamsRequest(ctx), ", :")+") RETURNING *")))
	if err != nil {
		return nil, err
	}

	err = stmt.Get(data, data)
	stmt.Close()

	return data, err
}

// Update data in target by id
// ID taken from passed data
func Update(ctx context.Context, conn *sqlx.DB, target string, data RequestParameter) (interface{}, error) {
	h := Get(ctx)
	query := h.BuildQuery(data)

	if conn == nil {
		return nil, db.ErrDBConnNotEstablished
	}

	_, err := conn.NamedExec(
		conn.Rebind(utils.JoinStrings(" ", "UPDATE "+target+" SET ", strings.Join(query, ", "),
			"WHERE id=:id")),
		data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type Helper struct {
	requestParamsCache map[string][]string
}

func Create(ctx context.Context) (context.Context, *Helper) {
	h := &Helper{}
	return context.WithValue(ctx, HelperItem, h), h
}

func Get(ctx context.Context) *Helper {
	if v := ctx.Value(HelperItem); v != nil {
		return v.(*Helper)
	}
	return nil
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

func (h *Helper) requestParams(res *[]string, data interface{}) {
	Types := reflect.TypeOf(data)
	Values := reflect.ValueOf(data)

	for i := 0; i < Types.NumField(); i++ {
		if Types.Field(i).Anonymous {
			h.requestParams(res, Values.Field(i).Interface())
		}
		*res = append(*res, Types.Field(i).Tag.Get("db"))
	}
}

// RequestParams returns sql request parameters for data
func (h *Helper) RequestParams(data interface{}) []string {
	if h.requestParamsCache == nil {
		h.requestParamsCache = make(map[string][]string)
	}

	typeName := reflect.TypeOf(data).String()
	if v, ok := h.requestParamsCache[typeName]; ok {
		return v
	}

	Values := reflect.ValueOf(data).Elem()
	res := make([]string, 0, Values.NumField())
	h.requestParams(&res, valueFromPtr(data))
	h.requestParamsCache[typeName] = res

	return res
}

// RequestParamsWithout returns sql request parameters for data without
// removed fields
func (h *Helper) RequestParamsWithout(data interface{}, remove ...string) []string {
	sqlPrams := h.RequestParams(data)
	res := make([]string, 0, len(sqlPrams))
	for _, v := range sqlPrams {
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

func (h *Helper) BuildQuery(data interface{}) []string {
	sqlPrams := h.RequestParams(data)
	query := make([]string, 0, len(sqlPrams))
	for _, param := range sqlPrams {
		query = append(query, param+"=:"+param)
	}
	return query
}
