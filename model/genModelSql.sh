goctl model mysql ddl \
  -src=./sql/genSql.sql \
  -dir=./genModel \
  -cache=false \
  --style=gozero \
  --home=../1.7.1