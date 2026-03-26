package db

type Db interface {
	CreateClient(connstr string) error
	Close() error
}
