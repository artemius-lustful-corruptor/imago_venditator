package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/Davincible/goinsta"
	"github.com/spf13/viper"
)

var checkedUser = make(map[string]bool)

var insta *goinsta.Instagram

func login() {
	err := reloadSession()
	if err != nil {
		createAndSaveSession()
	}
}

func reloadSession() error {
	insta, err := goinsta.Import("./insta-session")
	if err != nil {
		return errors.New("Couldn't recover the session")
	}

	if insta != nil {
		instabot.Insta = insta
	}

	log.Println("Successfully logged in")
	return nil
}

func createAndSaveSession() {
	insta := goinsta.New(
		viper.GetString("user.instagram.username"),
		viper.GetString("user.instagram.password"))
	instabot.Insta = insta

	err := instabot.Insta.Login()
	check(err)
	err = insta.Export("./insta-session")
	check(err)
	log.Println("Created and saved the session")
}

func (myInstabot Instabot) syncFollowers() {
	following := myInstabot.Insta.Account.Following()
	followers := myInstabot.Insta.Account.Followers()

	var followerUsers []goinsta.User
	var followingUsers []goinsta.User

	for following.Next() {
		for _, user := range following.Users {
			log.Println(*user)
			followingUsers = append(followingUsers, *user)
		}
	}

	for followers.Next() {
		for _, user := range followers.Users {
			followerUsers = append(followerUsers, *user)
		}
	}

	var users []goinsta.User
	for _, user := range followingUsers {
		if containsString(userWhitelist, user.Username) {
			continue
		}

		if !containsUser(followerUsers, user) {
			users = append(users, user)
		}
	}

	if len(users) == 0 {
		return
	}

	//fmt.Printf("\n%d users are not following you back!\n", len(users))
	//answer := getInput("Do you want to review these users ? [yN]")

	//if answer != "y" {
	//	fmt.Println("Not unfollowing.")
	//	os.Exit(0)
	//}

	//answerUnfollowAll := getInput("Unfollow everyone ? [yN]")

	for _, user := range users {
		//	if answerUnfollowAll != "y" {
		//	answerUserUnfollow := getInput("Unfollow %s ? [yN]", user.Username)
		//	if answerUserUnfollow != "y" {
		//		userWhitelist = append(userWhitelist, user.Username)
		//		continue
		//	}
		//}
		userBlacklist = append(userBlacklist, user.Username)
		user.Unfollow()
		time.Sleep(6 * time.Second)
	}
}

func (myInstabot Instabot) loopTags() {
	for tag = range tagsList {
		limitsConf := viper.GetStringMap("tags." + tag)
		limits = map[string]int{
			"follow":  int(limitsConf["follow"].(float64)),
			"like":    int(limitsConf["like"].(float64)),
			"comment": int(limitsConf["comment"].(float64)),
		}

		numFollowed = 0
		numLiked = 0
		numCommented = 0
		// They have share memory?
		myInstabot.browse()
	}
	buildReport()
}

func (myInstabot Instabot) goThrough(images *goinsta.FeedTag) {
	var i = 0

	for _, image := range images.Items {
		if numFollowed >= limits["follow"] && numLiked >= limits["like"] && numCommented >= limits["comment"] {
			break
		}

		if image.User.Username == viper.GetString("user.instagram.username") {
			continue
		}

		if i >= limits["follow"] && i >= limits["like"] && i >= limits["comment"] {
			break
		}

		if checkedUser[image.User.Username] {
			continue
		}

		var userInfo *goinsta.User
		err := retry(10, 20*time.Second, func() (err error) {
			userInfo, err = myInstabot.Insta.Profiles.ByName(image.User.Username)
			return
		})

		check(err)

		followerCount := userInfo.FollowerCount

		buildLine()

		checkedUser[userInfo.Username] = true
		log.Println("Checking followers for " + userInfo.Username + " - for #" + tag)
		log.Printf("%s has %d followers\n", userInfo.Username, followerCount)
		i++

		like := followerCount > likeLowerLimit && followerCount < likeUpperLimit && numLiked < limits["like"]
		follow := followerCount > followLowerLimit && followerCount < followUpperLimit && numFollowed < limits["follow"] && like
		comment := followerCount > commentLowerLimit && followerCount < commentUpperLimit && numCommented < limits["comment"] && like

		skip := false
		following := myInstabot.Insta.Account.Following() //????

		var followingUsers []goinsta.User
		for following.Next() {
			for _, user := range following.Users {
				followingUsers = append(followingUsers, *user)
			}
		}

		for _, user := range followingUsers {
			if user.Username == userInfo.Username {
				skip = true
				break
			}
		}

		if !skip {
			if like {
				myInstabot.likeImage(*image)
				if follow && !containsString(userBlacklist, userInfo.Username) {
					myInstabot.followUser(userInfo)
				}
				if comment {
					myInstabot.commentImage(*image)
				}
			}
		}

		log.Printf("%s done\n\n", userInfo.Username)

		time.Sleep(20 * time.Second)

	}

}

func (myInstabot Instabot) followUser(user *goinsta.User) {
	log.Printf("Following %s\n", user.Username)
	//err := user.FriendShip()
	//check(err)

	if !user.Friendship.Following {
		user.Follow()
		log.Println("Followed")
		numFollowed++
		report[line{tag, "follow"}]++
	} else {
		log.Println("Already following " + user.Username)
	}
}

func (myInstabot Instabot) likeImage(image goinsta.Item) {
	log.Println("Liking the picture")
	if !image.HasLiked {
		image.Like()
		log.Println("Liked")
		numLiked++
		report[line{tag, "like"}]++
	} else {
		log.Println("Image already liked")
	}
}

func (myInstabot Instabot) commentImage(image goinsta.Item) {
	rand.Seed(time.Now().Unix())
	text := commentsList[rand.Intn(len(commentsList))]
	comments := image.Comments
	if comments == nil {
		// What is it?
		newComments := goinsta.Comments{}
		rs := reflect.ValueOf(&newComments).Elem()
		rf := rs.FieldByName("item")
		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
		item := reflect.New(reflect.TypeOf(image))
		item.Elem().Set(reflect.ValueOf(image))
		rf.Set(item)
		newComments.Add(text)
	} else {
		comments.Add(text)
	}
	log.Println("Commented " + text)
	numCommented++
	report[line{tag, "comment"}]++ //what is line[{tag, comment}]
}

func (myInstabot Instabot) browse() {
	var i = 0
	for numFollowed < limits["follow"] || numLiked < limits["like"] || numCommented < limits["comment"] {
		log.Println("Fetching the list of images for #" + tag + "\n")
		i++

		var images *goinsta.FeedTag
		err := retry(10, 20*time.Second, func() (err error) {
			images, err = myInstabot.Insta.Feed.Tags(tag)
			return
		})
		check(err)

		myInstabot.goThrough(images)

		// ??? limits.maxRetry
		if viper.IsSet("limits.maxRetry") && i > viper.GetInt("limits.maxRetry") {
			log.Println("Currentrly not enough images for this tag to achieve goals")
			break
		}
	}
}

func containsString(slice []string, user string) bool {
	for _, currentUser := range slice {
		if currentUser == user {
			return true
		}
	}
	return false
}

func containsUser(slice []goinsta.User, user goinsta.User) bool {
	for _, currentUser := range slice {
		if currentUser.Username == user.Username {
			return true
		}
	}
	return false
}

func getInput(format string, args ...interface{}) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(format, args...)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

func (myInstabot Instabot) updateConfig() {
	viper.Set("whitelist", userWhitelist)
	viper.Set("blacklist", userBlacklist)

	err := viper.WriteConfig()
	if err != nil {
		log.Println("Update config file error", err)
		return
	}

	log.Println("Config file updated")
}
