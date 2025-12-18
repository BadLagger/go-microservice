package main

import (
	"go-microservice/utils"
)

func main() {
	log := utils.GlobalLogger().SetLevel(utils.Debug)
	log.Info("App start!!!")
	defer log.Info("App DONE!!!")
}
