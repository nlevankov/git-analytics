package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	git "github.com/go-git/go-git/v5"
)

type PersonInfo struct {
	Name         string
	AddedRows    int
	DeletedRows  int
	CommitsCount int
	//SigningKey   string
}

func main() {
	abortSpinner := make(chan struct{}, 1)
	go spinner(300*time.Millisecond, abortSpinner)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	repo, err := git.PlainOpen(path.Join(cwd, "repo"))
	if err != nil {
		log.Fatal(err)
	}
	toDate := time.Now()
	fromDate := toDate.AddDate(0, -12, 0)

	commitsIterator, err := repo.Log(&git.LogOptions{All: true, Since: &fromDate, Until: &toDate})
	if err != nil {
		log.Fatal(err)
	}

	personInfosByEmail := make(map[string]PersonInfo)
	err = commitsIterator.ForEach(func(commit *object.Commit) error {
		// email normalization - пока отбрасываем signingkey часть
		var normalizedEmail string
		emailWithoutSigningKey := regexp.MustCompile(`(.*?) signingkey .*`)
		found := emailWithoutSigningKey.FindSubmatch([]byte(commit.Author.Email))
		if len(found) == 2 {
			normalizedEmail = strings.TrimSpace(string(found[1]))
		} else {
			normalizedEmail = strings.TrimSpace(commit.Author.Email)
		}
		normalizedEmail = strings.ToLower(normalizedEmail)

		personInfoByEmail := personInfosByEmail[normalizedEmail]
		personInfoByEmail.Name = commit.Author.Name
		fileStats, err := commit.Stats()
		if err != nil {
			return err
		}
		for _, fileStat := range fileStats {
			personInfoByEmail.AddedRows += fileStat.Addition
			personInfoByEmail.DeletedRows += fileStat.Deletion
		}
		personInfoByEmail.CommitsCount++
		personInfosByEmail[normalizedEmail] = personInfoByEmail

		return nil
	})

	personInfosByName := make(map[string]PersonInfo)
	for _, personInfoByEmail := range personInfosByEmail {
		normalizedName := strings.ToLower(personInfoByEmail.Name)

		personInfoByName, ok := personInfosByName[normalizedName]
		if ok {
			personInfoByName.AddedRows += personInfoByEmail.AddedRows
			personInfoByName.DeletedRows += personInfoByEmail.DeletedRows
			personInfoByName.CommitsCount += personInfoByEmail.CommitsCount
			personInfosByName[normalizedName] = personInfoByName
		} else {
			personInfosByName[normalizedName] = personInfoByEmail
		}
	}

	abortSpinner <- struct{}{}
	fmt.Printf("\rFrom: %v\n", fromDate)
	fmt.Printf("To: %v\n\n", toDate)
	fmt.Printf("%v\t%v\t%v\t%v\n", "Name", "CommitsCount", "AddedRows", "DeletedRows")
	for _, personInfo := range personInfosByName {
		fmt.Printf("%v\t%v\t%v\t%v\n", personInfo.Name, personInfo.CommitsCount, personInfo.AddedRows, personInfo.DeletedRows)
	}
}

func spinner(delay time.Duration, abort <-chan struct{}) {
	for {
		for _, r := range `-\|/` {
			select {
			case <-abort:
				return
			default:
				fmt.Printf("\r%c Please wait %c", r, r)
				time.Sleep(delay)
			}
		}
	}
}
