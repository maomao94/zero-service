
func (m *default{{.upperStartCamelObject}}Model) Update(ctx context.Context,session sqlx.Session, {{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}})  (sql.Result,error) {
	newData.DeleteTime = time.Unix(0, 0)
    newData.DelState = 0
	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}})
        if err!=nil{
            return nil,err
        }
     {{end}}{{.keys}}
	return m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
	query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}})
	}
	return conn.ExecCtx(ctx, query, {{.expressionValues}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}})
	}
	return m.conn.ExecCtx(ctx, query, {{.expressionValues}}){{end}}
}

func (m *default{{.upperStartCamelObject}}Model) UpdateWithVersion(ctx context.Context,session sqlx.Session,{{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) error {

    {{if .containsIndexCache}}
     oldVersion := newData.Version
     newData.Version += 1
    {{else}}
    oldVersion := data.Version
    data.Version += 1
    {{end}}

	var sqlResult sql.Result
	var err error

	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}})
            if err!=nil{
                return err
            }
    {{end}}{{.keys}}
	sqlResult,err =  m.ExecCtx(ctx,func(ctx context.Context,conn sqlx.SqlConn) (result sql.Result, err error) {
	query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ? ", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}
	return conn.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ? ", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		sqlResult,err  =  session.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}else{
		sqlResult,err  =  m.conn.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}
	{{end}}
	if err != nil {
		return err
	}
	updateCount , err := sqlResult.RowsAffected()
	if err != nil{
		return err
	}
	if updateCount == 0 {
		return ErrNoRowsUpdate
	}

	return nil
}

func (m *default{{.upperStartCamelObject}}Model) DeleteSoft(ctx context.Context,session sqlx.Session,id int64) error {
    data, err := m.FindOne(ctx, id)
    if err != nil {
        return err
    }
    data.DelState = 1
    data.DeleteTime = time.Now()
	if err:= m.UpdateWithVersion(ctx,session, data);err!= nil{
		return errors.Wrapf(errors.New("delete soft failed "),"{{.upperStartCamelObject}}Model delete err : %+v",err)
	}
	return nil
}

func (m *default{{.upperStartCamelObject}}Model) FindSum(ctx context.Context,builder squirrel.SelectBuilder, field string) (float64,error) {

    if len(field) == 0 {
        return 0, errors.Wrapf(errors.New("FindSum Least One Field"), "FindSum Least One Field")
    }

    builder = builder.Columns("IFNULL(SUM(" + field + "),0)")

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}

	var resp float64
	{{if .withCache}}err = m.QueryRowNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindCount(ctx context.Context, builder squirrel.SelectBuilder, field string) (int64,error) {

    if len(field) == 0 {
        return 0, errors.Wrapf(errors.New("FindCount Least One Field"), "FindCount Least One Field")
    }

	builder = builder.Columns("COUNT(" + field + ")")

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}

	var resp int64
	{{if .withCache}}err = m.QueryRowNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindAll(ctx context.Context,builder squirrel.SelectBuilder,orderBy ...string) ([]*{{.upperStartCamelObject}},error) {

    builder = builder.Columns({{.lowerStartCamelObject}}Rows)

	if len(orderBy) == 0{
		builder = builder.OrderBy("id DESC")
	}else{
		builder = builder.OrderBy(orderBy...)
	}

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByPage(ctx context.Context,builder squirrel.SelectBuilder,page ,pageSize int64,orderBy ...string) ([]*{{.upperStartCamelObject}},error) {

    builder = builder.Columns({{.lowerStartCamelObject}}Rows)

	if len(orderBy) == 0{
		builder = builder.OrderBy("id DESC")
	}else{
		builder = builder.OrderBy(orderBy...)
	}

	if page < 1{
		page = 1
	}
	offset := (page - 1) * pageSize

	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}


func (m *default{{.upperStartCamelObject}}Model) FindPageListByPageWithTotal(ctx context.Context,builder squirrel.SelectBuilder,page ,pageSize int64, orderBy ...string) ([]*{{.upperStartCamelObject}},int64,error) {

    total, err := m.FindCount(ctx, builder, "id")
    if err != nil {
        return nil, 0, err
    }

    builder = builder.Columns({{.lowerStartCamelObject}}Rows)

	if len(orderBy) == 0{
		builder = builder.OrderBy("id DESC")
	}else{
		builder = builder.OrderBy(orderBy...)
	}

	if page < 1{
		page = 1
	}
	offset := (page - 1) * pageSize

	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil,total, err
	}

	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp,total, nil
	default:
		return nil,total, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByIdDESC(ctx context.Context,builder squirrel.SelectBuilder ,preMinId ,pageSize int64) ([]*{{.upperStartCamelObject}},error) {

    builder = builder.Columns({{.lowerStartCamelObject}}Rows)

	if preMinId > 0 {
		builder = builder.Where(" id < ? " , preMinId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id DESC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByIdASC(ctx context.Context,builder squirrel.SelectBuilder,preMaxId ,pageSize int64) ([]*{{.upperStartCamelObject}},error)  {

    builder = builder.Columns({{.lowerStartCamelObject}}Rows)

	if preMaxId > 0 {
		builder = builder.Where(" id > ? " , preMaxId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id ASC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx,&resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx,&resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) Trans(ctx context.Context,fn func(ctx context.Context,session sqlx.Session) error) error {
	{{if .withCache}}
	return m.TransactCtx(ctx,func(ctx context.Context,session sqlx.Session) error {
		return fn(ctx,session)
	})
	{{else}}
	return m.conn.TransactCtx(ctx,func(ctx context.Context,session sqlx.Session) error {
		return fn(ctx,session)
	})
	{{end}}
}

func(m *default{{.upperStartCamelObject}}Model)  SelectBuilder() squirrel.SelectBuilder {
	return squirrel.Select().From(m.table)
}