package dependencies

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Dependencies struct {
	Logger     *logrus.Logger
	DB         *gorm.DB
	RabbitConn *amqp.Connection
	RabbitCh   *amqp.Channel
}
