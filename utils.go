package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	nomail      bool
	run         bool
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
	flag.Parse()
}

func getConfig() {
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
	// shared memory????
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
	//fmt.Sprintf("The script has stopped due to an unecoverable error :\n%s", err), false
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

	filename := "report.txt"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(reportAsString); err != nil {
		panic(err)
	}
	fmt.Println("Report has been added")

}

