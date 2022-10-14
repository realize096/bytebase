package mysql

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/mysql"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustUnsignedAdvisor)(nil)
	_ ast.Visitor     = (*columnAutoIncrementMustUnsignedChecker)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLAutoIncrementColumnMustUnsigned, &ColumnAutoIncrementMustUnsignedAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLAutoIncrementColumnMustUnsigned, &ColumnAutoIncrementMustUnsignedAdvisor{})
}

// ColumnAutoIncrementMustUnsignedAdvisor is the advisor checking for unsigned auto-increment column.
type ColumnAutoIncrementMustUnsignedAdvisor struct {
}

// Check checks for unsigned auto-increment column.
func (*ColumnAutoIncrementMustUnsignedAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmtList, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnAutoIncrementMustUnsignedChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.OriginTextPosition()
		(stmt).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type columnAutoIncrementMustUnsignedChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (checker *columnAutoIncrementMustUnsignedChecker) Enter(in ast.Node) (ast.Node, bool) {
	var columnList []columnData
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		for _, column := range node.Cols {
			if !autoIncrementColumnIsUnsigned(column) {
				columnList = append(columnList, columnData{
					table:  node.Table.Name.O,
					column: column.Name.Name.O,
					line:   column.OriginTextPosition(),
				})
			}
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					if !autoIncrementColumnIsUnsigned(column) {
						columnList = append(columnList, columnData{
							table:  node.Table.Name.O,
							column: column.Name.Name.O,
							line:   node.OriginTextPosition(),
						})
					}
				}
			case ast.AlterTableChangeColumn, ast.AlterTableModifyColumn:
				if !autoIncrementColumnIsUnsigned(spec.NewColumns[0]) {
					columnList = append(columnList, columnData{
						table:  node.Table.Name.O,
						column: spec.NewColumns[0].Name.Name.O,
						line:   node.OriginTextPosition(),
					})
				}
			}
		}
	}

	for _, column := range columnList {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.AutoIncrementColumnSigned,
			Title:   checker.title,
			Content: fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", column.table, column.column),
			Line:    checker.line,
		})
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*columnAutoIncrementMustUnsignedChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func autoIncrementColumnIsUnsigned(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionAutoIncrement && !mysql.HasUnsignedFlag(column.Tp.GetFlag()) {
			return false
		}
	}
	return true
}
