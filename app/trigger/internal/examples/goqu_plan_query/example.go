//go:build ignore

// Package goqu_plan_query 为「goqu + dbx」查询 plan 表的参考示例，默认不参与 go build ./...
//
// 使用方式：
//  1. 复制需要的片段到业务包；或
//  2. 在本目录执行：go run -tags ignore .   （需去掉本文件第一行 build tag 临时调试）
//
// 依赖与 trigger 服务一致：zero-service/common/dbx、goqu v9。
package goqu_plan_query

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"zero-service/common/dbx"
)

// SelectEnabledPlansSample 演示：从 plan 表查询若干启用状态记录，打印 SQL 与结果（参考用）。
func SelectEnabledPlansSample(ctx context.Context, dsn string) error {
	db, err := dbx.NewQoqu(dsn)
	if err != nil {
		return err
	}

	query := db.From("plan").
		Select("id", "plan_id", "plan_name").
		Where(goqu.C("status").Eq(1)).
		Where(goqu.C("del_state").Eq(0)).
		Limit(5)

	sqlStr, args, err := query.ToSQL()
	if err != nil {
		return fmt.Errorf("goqu ToSQL: %w", err)
	}
	fmt.Println("sql:", sqlStr)
	fmt.Println("args:", args)

	type planRow struct {
		Id       int64  `db:"id"`
		PlanId   string `db:"plan_id"`
		PlanName string `db:"plan_name"`
	}
	var rows []planRow
	if err := query.ScanStructsContext(ctx, &rows); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	for i, p := range rows {
		fmt.Printf("row %d: id=%d plan_id=%s plan_name=%s\n", i, p.Id, p.PlanId, p.PlanName)
	}
	return nil
}
