package main

import (
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	dev         bool
	nomail      bool
	run         bool
	logs        bool
	noduplicate bool
	unfollow    bool
)

var likeLowerLimit int
var likeUpperLimit int

var followLowerLimit int
var followUpperLimit int

var commentLowerLimit int
var commentUpperLimit int

var tagsList map[string]interface{}

var limits map[string]int

var commentsList []string

type line struct {
	Tag, Action string
}

var report map[line]int

var userBlacklist []string
var userWhitelist []string

var numFollowed int
var numCommented int
var numLiked int

var tag string

func check(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

func parseOptions() {
	flag.BoolVar(&run, "run", false, "Use this option to follow, like and comment")
	flag.BoolVar(&unfollow, "sync", false, "Use this option to unfollow those who are not following back")
	flag.BoolVar(&nomail, "nomail", false, "Use this option to disable the email notifications")
	flag.BoolVar(&dev, "dev", false, "Use this option to use the script in development")
	flag.BoolVar(&logs, "logs", false, "Use this option to enable the logfile")
	flag.BoolVar(&noduplicate, "noduplicate", false, "Use this option to skip following, liking, and commenting same user in this session")

	flag.Parse()

	if logs {
		// t := time.Now()
		// logFile, err := os.OpenFile("instabot-"+t.Format("2006-01-02-15-04-05")+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		// check(err)
		// defer logFile.Close()
		// mw := io.MultiWriter(os.Stdout, logFile)
		// log.SetOutput(mw)
	}
}

func getConfig() {
	//folder := "config"
	//if dev {
	//	folder = "local"
	//}
	viper.SetConfigFile("./config.json")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	viper.SetEnvPrefix("instabot")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	log.Printf("Using config: %s\n\n", viper.ConfigFileUsed())

	likeLowerLimit = viper.GetInt("limits.like.min")
	likeUpperLimit = viper.GetInt("limits.like.max")

	followLowerLimit = viper.GetInt("limits.comment.min")
	followUpperLimit = viper.GetInt("limits.comment.max")

	commentLowerLimit = viper.GetInt("limits.comment.min")
	commentUpperLimit = viper.GetInt("limits.comment.max")

	tagsList = viper.GetStringMap("tags")
	commentsList = viper.GetStringSlice("comments")

	userBlacklist = viper.GetStringSlice("blacklist")
	userWhitelist = viper.GetStringSlice("whitelist")

	// type Report struct {
	// Tag, Action string
	// }
	report = make(map[line]int)
}

func send(body string, success bool) {
	if !nomail {
		from := viper.GetString("user.mail.from")
		pass := viper.GetString("user.mail.password")
		to := viper.GetString("user.mail.to")

		status := func() string {
			if success {
				return "Success!"
			}
			return "Failure!"
		}()

		msg := "From: " + from + "\n" +
			"To:" + to + "\n" +
			"Subject:" + status + " go-instabot\n\n" +
			body

		err := smtp.SendMail(viper.GetString("user.mail.smtp"),
			smtp.PlainAuth("", from, pass, viper.GetString("user.mail.server")),
			from, []string{to}, []byte(msg))

		if err != nil {
			log.Printf("smtp error: %s", err)
			return
		}

		log.Print("sent")
	}
}

func retry(maxAttempts int, sleep time.Duration, function func() error) (err error) {
	for currentAttempt := 0; currentAttempt < maxAttempts; currentAttempt++ {
		err = function()
		if err == nil {
			return
		}
		for i := 0; i <= currentAttempt; i++ {
			time.Sleep(sleep)
		}
		log.Println("Retrying after error:", err)
	}
	send(fmt.Sprintf("The script has stopped due to an unecoverable error :\n%s", err), false)
	return fmt.Errorf("After %d attempts, last error: %s", maxAttempts, err)
}

func buildLine() {
	reportTag := ""
	// ?????
	for index, element := range report {
		if index.Tag == tag {
			reportTag += fmt.Sprintf("%s %d/%d - ", index.Action, element, limits[index.Action])
		}
	}
	if reportTag != "" {
		log.Println(strings.TrimSuffix(reportTag, " - "))
	}
}

func buildReport() {
	reportAsString := ""
	for index, element := range report {
		var times string
		if element == 1 {
			times = "time"
		} else {
			times = "times"
		}
		if index.Action == "like" {
			reportAsString += fmt.Sprintf("#%s has been liked %d %s \n", index.Tag, element, times)
		} else {
			reportAsString += fmt.Sprintf("#%s has been %sed %d %s\n", index.Tag, index.Action, element, times)
		}
	}

	fmt.Println(reportAsString)

	send(reportAsString, true)

}
