
package main

import 	"github.com/Davincible/goinsta"

type Instabot struct {
	Insta *goinsta.Instagram
}

var instabot Instabot

func main() {
	parseOptions()
	getConfig()
	login()
	if unfollow {
		instabot.syncFollowers()
	} else if run {
		instabot.loopTags()
	}

	instabot.updateConfig()
}
