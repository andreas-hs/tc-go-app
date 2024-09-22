package dependencies

import (
	"github.com/andreas-hs/tc-go-app/internal/infrastructure/database"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Dependencies struct {
	Logger     *logrus.Logger
	DB         database.Database
	RabbitConn *amqp.Connection
	RabbitCh   *amqp.Channel
}
