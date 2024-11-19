Update(ctx context.Context,session sqlx.Session, data *{{.upperStartCamelObject}}) (sql.Result, error)
UpdateWithVersion(ctx context.Context,session sqlx.Session,data *{{.upperStartCamelObject}}) error
Trans(ctx context.Context,fn func(context context.Context,session sqlx.Session) error) error
SelectBuilder() squirrel.SelectBuilder
DeleteSoft(ctx context.Context,session sqlx.Session, data *{{.upperStartCamelObject}}) error
FindSum(ctx context.Context,sumBuilder squirrel.SelectBuilder,field string) (float64,error)
FindCount(ctx context.Context,countBuilder squirrel.SelectBuilder,field string) (int64,error)
FindAll(ctx context.Context,rowBuilder squirrel.SelectBuilder,orderBy string) ([]*{{.upperStartCamelObject}},error)
FindPageListByPage(ctx context.Context,rowBuilder squirrel.SelectBuilder,page ,pageSize int64,orderBy string) ([]*{{.upperStartCamelObject}},error)
FindPageListByPageWithTotal(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*{{.upperStartCamelObject}}, int64, error)
FindPageListByIdDESC(ctx context.Context,rowBuilder squirrel.SelectBuilder ,preMinId ,pageSize int64) ([]*{{.upperStartCamelObject}},error)
FindPageListByIdASC(ctx context.Context,rowBuilder squirrel.SelectBuilder,preMaxId ,pageSize int64) ([]*{{.upperStartCamelObject}},error)