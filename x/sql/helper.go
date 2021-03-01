package sql

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/utils"
)

type RequestParameter interface {
	SQLParamsRequest() []string
}

func SelectByID(conn *sqlx.DB, target string, id int64, data interface{}) error {
	if conn == nil {
		return db.ErrDBConnNotEstablished
	}

	err := conn.Get(data, conn.Rebind(utils.JoinStrings(" ", "SELECT *", target, "WHERE id=$1")), id)
	if err != nil {
		return err
	}

	return nil
}

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

func InsertInto(conn *sqlx.DB, target string, data RequestParameter) (interface{}, error) {
	if conn == nil {
		return nil, db.ErrDBConnNotEstablished
	}

	stmt, err := conn.PrepareNamed(
		conn.Rebind(utils.JoinStrings(" ", "INSERT INTO", target, "("+strings.Join(data.SQLParamsRequest(), ", ")+")",
			"VALUES", "("+":"+strings.Join(data.SQLParamsRequest(), ", :")+") RETURNING *")))
	if err != nil {
		return nil, err
	}

	err = stmt.Get(data, data)
	stmt.Close()

	return data, err
}

func Update(conn *sqlx.DB, target string, writeData RequestParameter) (interface{}, error) {
	query := make([]string, 0, len(writeData.SQLParamsRequest()))
	for _, param := range writeData.SQLParamsRequest() {
		query = append(query, param+"=:"+param)
	}

	if conn == nil {
		return nil, db.ErrDBConnNotEstablished
	}

	_, err := conn.NamedExec(
		conn.Rebind(utils.JoinStrings(" ", "UPDATE "+target+" SET ", strings.Join(query, ", "),
			"WHERE id=:id")),
		writeData)
	if err != nil {
		return nil, err
	}

	return writeData, nil
}
