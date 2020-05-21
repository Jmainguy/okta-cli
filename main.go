package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/okta/okta-sdk-golang/okta"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func resetPasword(user *okta.User) {
	oktaurl := os.Getenv("OKTAURL")
	oktatoken := os.Getenv("OKTATOKEN")

	url := fmt.Sprintf("%s/api/v1/users/%s/lifecycle/reset_password?sendEmail=true", oktaurl, user.Id)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", oktatoken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Printf("Sent %s password reset email\n", (*user.Profile)["secondEmail"])
}

func updateProfile(user *okta.User, client *okta.Client, profile *okta.UserProfile, groupMap map[string]string) {
	updateNeeded := false
	if (*user.Profile)["firstName"] != (*profile)["firstName"] {
		updateNeeded = true
	} else if (*user.Profile)["lastName"] != (*profile)["lastName"] {
		updateNeeded = true
	} else if (*user.Profile)["secondEmail"] != (*profile)["secondEmail"] {
		updateNeeded = true
	}
	if updateNeeded {
		fmt.Println("Updating user profile")
		updatedUser := &okta.User{
			Profile: profile,
		}
		groupId := groupMap["Coke"]
		user, _, err := client.User.UpdateUser(user.Id, *updatedUser, nil)
		if err != nil {
			panic(err)
		}
		_, err = client.Group.AddUserToGroup(groupId, user.Id)
		if err != nil {
			panic(err)
		}
	}
}

func getGroupMap(client *okta.Client) map[string]string {
	groupMap := make(map[string]string)
	groups, resp, err := client.Group.ListGroups(nil)
	if err != nil {
		fmt.Println(err)
		fmt.Println(resp)
	}
	for _, group := range groups {
		groupMap[group.Profile.Name] = group.Id
	}
	return groupMap
}

func getUser(shortname string) (user *okta.User) {
	oktaurl := os.Getenv("OKTAURL")
	oktatoken := os.Getenv("OKTATOKEN")

	url := fmt.Sprintf("%s/api/v1/users/%s", oktaurl, shortname)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", oktatoken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	err = json.Unmarshal([]byte(body), &user)
	if err != nil {
		fmt.Println(err)
	}
	return user
}

func createUser(client *okta.Client, groupMap map[string]string, email, first, last string, groups []string) (userID string) {
	p := &okta.PasswordCredential{
		Value: "kurtwasacoward",
	}
	uc := &okta.UserCredentials{
		Password: p,
	}
	email = strings.ToLower(email)
	first = strings.Title(first)
	last = strings.Title(last)
	username := strings.Split(email, "@")[0]
	var login string
	login = username + "@example.com"
	groups = append(groups, "Everyone")

	profile := okta.UserProfile{}
	profile["firstName"] = first
	profile["lastName"] = last
	profile["secondEmail"] = email
	profile["email"] = login
	profile["login"] = login

	u := &okta.User{
		Credentials: uc,
		Profile:     &profile,
	}

	user, _, err := client.User.CreateUser(*u, nil)
	if err != nil {
		// Check if err means user already exists
		if strings.Contains(err.Error(), "login: An object with this field already exists in the current organization") {
			// User already exists, grab their userID
			user = getUser(username)
			if user.Id == "" {
				fmt.Printf("Unable to find ID for %s\n", username)
				os.Exit(1)
			}
		} else {
			panic(err)
		}
	} else {
		// User created for first time, send them password reset email
		resetPasword(user)
	}
	// Groups user is currently in
	actualGroups, _, _ := client.User.ListUserGroups(user.Id, nil)
	// Groups user should be in
	desiredGroups := make(map[string]bool)
	for _, group := range groups {
		desiredGroups[group] = true
	}
	for _, ag := range actualGroups {
		if !desiredGroups[ag.Profile.Name] {
			// Remove any unwanted groups
			_, err = client.Group.RemoveGroupUser(ag.Id, user.Id)
			if err != nil {
				panic(err)
			}
		} else {
			// One less group to add user to later, since they are already in it
			delete(desiredGroups, ag.Profile.Name)
		}
	}
	// Add to any remaining groups
	for dg, _ := range desiredGroups {
		groupId := groupMap[dg]
		_, err = client.Group.AddUserToGroup(groupId, user.Id)
		if err != nil {
			panic(err)
		}
	}
	// Ensure Profile matches
	updateProfile(user, client, &profile, groupMap)

	return user.Id
}

func readPlay(filename string) (users oktaUsers) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &users)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return users
}

func main() {
	oktaurl := os.Getenv("OKTAURL")
	oktatoken := os.Getenv("OKTATOKEN")
	client, err := okta.NewClient(context.Background(), okta.WithOrgUrl(oktaurl), okta.WithToken(oktatoken), okta.WithCache(false))
	if err != nil {
		panic(err)
	}
	filename := flag.String("playbook", "/tmp/example.yaml", "playbook to run")
	flag.Parse()
	oktaUsers := readPlay(*filename)
	groupMap := getGroupMap(client)
	for _, user := range oktaUsers {
		userID := createUser(client, groupMap, user.Email, user.First, user.Last, user.Groups)
		fmt.Printf("User %s %s: UserID: %s\n", user.First, user.Last, userID)
		userGroups, _, err := client.User.ListUserGroups(userID, nil)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("  Groups:")
		for _, group := range userGroups {
			fmt.Printf("    %s\n", group.Profile.Name)
		}
	}
}
