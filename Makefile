all:
	go build

push: commit
	git push -f heroku main

tail: push
	heroku logs -t

commit:
	git add . && git commit -m "logdrain messages handling"

createschema:
	heroku pg:psql < schema.sql

inspect:
	echo "select count(*) from logs; select * from logs order by id desc limit 15;" | heroku pg:psql