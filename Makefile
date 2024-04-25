all:
	go build

deploy: all
	git push -f heroku main
	heroku logs -t

commit: all
	git add . && git commit -m "logdrain messages handling"

createschema:
	heroku pg:psql < schema.sql

inspect:
	echo "select count(*) from logs; select * from logs order by id desc limit 15;" | heroku pg:psql