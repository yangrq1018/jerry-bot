package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/thoas/go-funk"
	"github.com/urfave/cli/v2"
	"github.com/yangrq1018/jerry-bot/nintendo"
	"github.com/yangrq1018/jerry-bot/util"
)

func uploadBattle() *cli.Command {
	var (
		dryRun    bool
		anonymous bool
	)
	return &cli.Command{
		Name:    "upload",
		Aliases: []string{"u"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "cookie",
			},
			&cli.BoolFlag{
				Name: "new-cookie",
			},
			&cli.IntFlag{
				Name:    "battle-number",
				Aliases: []string{"bn"},
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Aliases:     []string{"d"},
				Destination: &dryRun,
			},
			&cli.BoolFlag{
				Name:        "anonymous",
				Destination: &anonymous,
			},
		},
		Action: func(c *cli.Context) error {
			cookie := c.String("cookie")
			if c.Bool("new-cookie") {
				var err error
				_, cookie, err = nintendo.GenNewCookie(os.Stdin)
				if err != nil {
					return err
				}
				log.Printf("cookie generated: %s", cookie)
			}
			splatoon := nintendo.NewClient(cookie).Splatoon()
			ink := nintendo.StatInk{
				Mode:      "cli",
				Splatoon:  splatoon,
				Anonymous: anonymous,
			}
			battles, err := splatoon.Recent50Battles()
			if err != nil {
				return err
			}

			bn := c.Int("battle-number")
			uploaded, err := nintendo.GetStatinkBattles()
			if err != nil {
				return err
			}

			var uploadCount int
			for n := len(battles) - 1; n >= 0; n-- {
				if bn != 0 && battles[n].BattleNumber != strconv.Itoa(bn) {
					continue
				}
				// check stat.ink uploaded
				if funk.ContainsInt(uploaded, util.MustAtoi(battles[n].BattleNumber)) {
					continue
				}
				fmt.Printf("uploading battle %s@%s: ", battles[n].BattleNumber, battles[n].PlayerResult.Player.PrincipalID)
				err = ink.PostBattleToStatink(battles[n], n == 0, dryRun)
				if err != nil {
					fmt.Printf("%v", err)
				} else {
					fmt.Printf("ok!")
					uploadCount++
				}
				fmt.Println()
			}
			if uploadCount == 0 {
				fmt.Println("battles are all clear")
			}
			return nil
		},
	}
}

func App() *cli.App {
	app := cli.NewApp()
	app.Commands = []*cli.Command{
		uploadBattle(),
	}
	return app
}

func main() {
	err := App().Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
