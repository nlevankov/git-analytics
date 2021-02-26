package gitanalytics

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/go-git/go-git/v5"
)

type PersonInfo struct {
	Name         string
	AddedRows    int
	DeletedRows  int
	CommitsCount int
	//SigningKey   string
}

// returns map[PersonName string]PersonInfo
func GetContributionInfos(pathToRepo string, fromDate, toDate time.Time) (map[string]PersonInfo, error) {
	repo, err := git.PlainOpen(pathToRepo)
	if err != nil {
		return nil, fmt.Errorf("git.PlainOpen (%v): %w", pathToRepo, err)
	}

	commitsIterator, err := repo.Log(&git.LogOptions{All: true, Since: &fromDate, Until: &toDate})
	if err != nil {
		return nil, fmt.Errorf("repo.Log: %w", err)
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
			return fmt.Errorf("commit.Stats: %w", err)
		}
		for _, fileStat := range fileStats {
			personInfoByEmail.AddedRows += fileStat.Addition
			personInfoByEmail.DeletedRows += fileStat.Deletion
		}
		personInfoByEmail.CommitsCount++
		personInfosByEmail[normalizedEmail] = personInfoByEmail

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("commitsIterator.ForEach: %w", err)
	}

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

	return personInfosByName, nil
}
