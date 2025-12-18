package repository

/*import (
	"context"
	"database/sql"
	"go-microservice/utils"
	"time"

	_ "github.com/lib/pq"
)

type PsqlRepository struct {
	db  *sql.DB
	log *utils.Logger
	cfg *DbCfg
}

type DbCfg struct {
	Host       string
	Port       string
	User       string
	Password   string
	Name       string
	CtxSecTout int
	SslMode    string
}

func PgConfigFromConfig(cfg *utils.Config) *DbCfg {
	sslModeStr := "disable"
	if cfg.DbSslMode {
		sslModeStr = "enable"
	}
	return &DbCfg{
		Host:       cfg.DbHost,
		Port:       cfg.DbPort,
		User:       cfg.DbUsername,
		Password:   cfg.DbPassword,
		Name:       cfg.DbName,
		CtxSecTout: cfg.DbCtxTimeoutSec,
		SslMode:    sslModeStr,
	}
}

func (cfg *DbCfg) String() string {
	return "host=" + cfg.Host +
		" port=" + cfg.Port +
		" user=" + cfg.User +
		" password=" + cfg.Password +
		" dbname=" + cfg.Name +
		" sslmode=" + cfg.SslMode
}

func NewPsqlRepository(ctx context.Context, dbCfg *DbCfg) *PsqlRepository {
	log := utils.GlobalLogger()
	log.Info("Try to create Postgres connection...")

	connCtx, cancel := context.WithTimeout(ctx, time.Duration(dbCfg.CtxSecTout)*time.Second)
	defer cancel()

	cfgStr := dbCfg.String()
	log.Debug("Repo cfg string: %s", cfgStr)
	db, err := sql.Open("postgres", cfgStr)
	if err != nil {
		log.Critical("Cann't connect to db: %+v", err)
		return nil
	}

	if err := db.PingContext(connCtx); err != nil {
		log.Critical("Cann't ping db")
		log.Critical("DbError: %+v", err)
		db.Close()
		return nil
	}

	log.Info("Postgres connection OK!")
	return &PsqlRepository{
		db:  db,
		log: log,
		cfg: dbCfg,
	}
}

func (r *PsqlRepository) Close() error {
	r.log.Info("Closing DB!")
	return r.db.Close()
}*/
