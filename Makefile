# refresh
r:
	@cls
	go run .

# Restart
R:
	@cls
	@echo Deleting server_database.db ...
	@del server_database.db
	go run .

open_db:
	@sqlite3 server_database.db

grpc_out:
	protoc ./proto/*.proto --go_out=. --go-grpc_out=.
