Update(ctx context.Context,session sqlx.Session, data *{{.upperStartCamelObject}}) (sql.Result, error)
UpdateWithVersion(ctx context.Context,session sqlx.Session, data *{{.upperStartCamelObject}}) error
Trans(ctx context.Context,fn func(ctx context.Context, session sqlx.Session) error) error
ExecCtx(ctx context.Context, session sqlx.Session, query string, args ...any) (sql.Result, error)
SelectWithBuilder(ctx context.Context, builder squirrel.SelectBuilder) ([]*{{.upperStartCamelObject}}, error)
SelectOneWithBuilder(ctx context.Context, builder squirrel.SelectBuilder) (*{{.upperStartCamelObject}}, error)
InsertWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.InsertBuilder) (sql.Result, error)
UpdateWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.UpdateBuilder) (sql.Result, error)
DeleteWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.DeleteBuilder) (sql.Result, error)
SelectBuilder() squirrel.SelectBuilder
InsertBuilder() squirrel.InsertBuilder
UpdateBuilder() squirrel.UpdateBuilder
DeleteBuilder() squirrel.DeleteBuilder
DeleteSoft(ctx context.Context, session sqlx.Session, id int64) error
FindSum(ctx context.Context, sumBuilder squirrel.SelectBuilder, field string) (float64,error)
FindCount(ctx context.Context, countBuilder squirrel.SelectBuilder, field string) (int64,error)
FindAll(ctx context.Context, rowBuilder squirrel.SelectBuilder, orderBy ...string) ([]*{{.upperStartCamelObject}},error)
FindPageListByPage(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy ...string) ([]*{{.upperStartCamelObject}},error)
FindPageListByPageWithTotal(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy ...string) ([]*{{.upperStartCamelObject}}, int64, error)
FindPageListByIdDESC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*{{.upperStartCamelObject}},error)
FindPageListByIdASC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*{{.upperStartCamelObject}},error)