package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/WLM1ke/gomoex"
)

func main() {

	// testReq()
	// return

	cl := gomoex.NewISSClient(http.DefaultClient)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()

	market := gomoex.MarketBonds
	// market := gomoex.MarketShares
	board := "TQCB"
	BoardSecurities, err := cl.BoardSecurities(ctx, gomoex.EngineStock, market, board)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("result:")
	for _, v := range BoardSecurities {
		fmt.Println(v)
	}

	// https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities.xml
}

func testReq() {

	url := "https://iss.moex.com/iss/engines/stock/markets/bonds/securities?marketprice_board=1"

	fmt.Println("Client start")
	responce, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(responce.StatusCode)

}

func dividends(cl *gomoex.ISSClient, security string) {

	// security = "LKOH"
	divs, err := cl.Dividends(context.Background(), security)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, div := range divs {
		fmt.Println(div)
	}

}
