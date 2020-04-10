/*
// TODO:
// 1. tests
// 2. consider having exchange interface, signal provider interface
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	util "github.com/CheshireCatNick/crypto-flash/pkg/util"
	"time"
	character "github.com/CheshireCatNick/crypto-flash/pkg/character"
	exchange "github.com/CheshireCatNick/crypto-flash/pkg/exchange"
	"sync"
)
const version = "3.2.4-beta"
const update = "1. Implement stop, take profit and trailing stop.\n" + 
	"2. Calculate win rate for strategy adjustment.\n" + 
	"3. FTX supports candle query more than 5000"
const tag = "Crypto Flash"

type user struct {
	Name string
	Key string
	Secret string
	SubAccount string
}
type lineConfig struct {
	Channel_Secret string
	Channel_Access_Token string
}
type config struct {
	Mode string
	Notify bool
	Users []user
	Line lineConfig
	Telegram string
}

func loadConfig(fileName string) config {
	var c config
	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		util.Error(tag, err.Error())
	}
	json.Unmarshal(bytes, &c)
	return c
}

func main() {
	var wg sync.WaitGroup
	config := loadConfig("config.json")
	fmt.Printf("Crypto Flash v%s initialized. Update: \n%s\n", version, update)
	
	var n *character.Notifier
	if config.Notify && config.Mode != "backtest" {
		n = character.NewNotifier(config.Line.Channel_Secret, 
			config.Line.Channel_Secret, config.Telegram)
		wg.Add(1)
		go n.Listen()
		n.Broadcast(tag, 
			fmt.Sprintf("Crypto Flash v%s initialized. Update: \n%s", 
				version, update))
	} else {
		n = nil
	}
	ftx := exchange.NewFTX("", "", "")
	sp := character.NewResTrend(ftx, n)
	if config.Mode == "trade" {
		for _, user := range config.Users {
			ftx := exchange.NewFTX(user.Key, user.Secret, user.SubAccount)
			trader := character.NewTrader(user.Name, ftx, n)
			signalChan := make(chan *util.Signal)
			sp.SubSignal(signalChan)
			wg.Add(1)
			go trader.Start(signalChan)
		}
		wg.Add(1)
		go sp.Start()
	} else if config.Mode == "notify" {
		wg.Add(1)
		go sp.Start()
	} else if config.Mode == "backtest" {
		//endTime, _ := time.Parse(time.RFC3339, "2019-12-01T05:00:00+00:00")
		endTime := time.Now()
		d := util.Duration{ Day: -20 }
		startTime := endTime.Add(d.GetTimeDuration())
		roi := sp.Backtest(startTime.Unix(), endTime.Unix())
		annual := util.CalcAnnualFromROI(roi, -d.GetTimeDuration().Seconds())
		fmt.Printf("Annual: %.2f%%", annual * 100)
	}
	wg.Wait()
}
