package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const commandFile = "./slack/commands.json"

// LoadCommands loads the commands from a JSON file
func LoadUrlNames() error {
	file, err := os.Open(commandFile)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file does not exist, return without error
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&urlList); err != nil {
		return err
	}
	return nil
}

// SaveCommands saves the commands to a JSON file
func saveUrlCommand() error {
	file, err := os.Create(commandFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(urlList)
}

type UrlCommand struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var urlList []UrlCommand

// ValidateCommand validates the command text received from Slack.
func validateCommand(command string) error {
	// Example validation: check if the command is not empty and follows a specific pattern
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Add more validation rules as needed
	if !strings.HasPrefix(command, "/") {
		return fmt.Errorf("command must start with a '/'")
	}

	return nil
}

func GoCommand(c echo.Context) error {
	text := c.FormValue("text")
	command := c.FormValue("command")
	// Validate the command
	commandErr := validateCommand(command)
	if commandErr != nil {
		return c.String(http.StatusOK, "Please provide a valid command")
	}

	return showGoCommand(text, c)
}

func showGoCommand(text string, c echo.Context) error {
	command := findUrlName(text)

	if command != nil {
		// Return the command URL if found
		return c.JSON(http.StatusOK, map[string]interface{}{
			"blocks": []interface{}{
				linkBlock(command.URL, command.Name),
			},
		})
	}

	// Look for partial matches in command names
	var matchingUrlName []UrlCommand
	for _, cmd := range urlList {
		nameParts := strings.Split(cmd.Name, "-")
		if len(nameParts) > 0 && nameParts[0] == text {
			matchingUrlName = append(matchingUrlName, cmd)
		}
	}

	// If partial matches are found return them
	if len(matchingUrlName) > 0 {
		// TODO: Add a final block
		return c.String(http.StatusOK, fmt.Sprintln("Multiple urls found!"))

	}

	// Return a more descriptive message if no command is found
	//TODO: Add a final block and sugest to call list command
	return c.String(http.StatusOK, fmt.Sprintf("No command found with the name '%s'. Please check the name and try again.", text))
}

func AddCommand(c echo.Context) error {
	command := c.FormValue("command")
	textName, textUrl, err := getNameAndUrl(c)

	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// Validate the command
	err = validateCommand(command)
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// Make sure command name does not exist
	urlNameExist := findUrlName(textName)

	if urlNameExist != nil {
		return c.String(http.StatusOK, "This url name already exist")
	}

	// Validate the URL
	isUrlValid := validateURL(textUrl)

	if !isUrlValid {
		return c.String(http.StatusOK, "Url is not valid")
	}

	// Add the textName + textURL to the list of commands in a folder
	urlList = append(urlList, UrlCommand{Name: textName, URL: textUrl})

	// Save the commands to the file
	if err := saveUrlCommand(); err != nil {
		return c.String(http.StatusOK, "Failed to save your command, please try again later")
	}

	return c.String(http.StatusOK, fmt.Sprintf("%s added with URL %s", textName, textUrl))
}

func EditCommand(c echo.Context) error {
	command := c.FormValue("command")
	textName, textUrl, err := getNameAndUrl(c)

	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// Validate the command
	err = validateCommand(command)
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// Make sure command name does not exist
	urlNameExist := findUrlName(textName)

	if urlNameExist == nil {
		return c.String(http.StatusOK, fmt.Sprintf("The command name '%s' was not found. Please make sure the name exists or add it first.", textName))
	}

	// Validate the URL
	isUrlValid := validateURL(textUrl)

	if !isUrlValid {
		return c.String(http.StatusOK, "Url is not valid")
	}

	// Edit the url command
	for i := range urlList {
		if urlList[i].Name == textName {
			urlList[i].URL = textUrl
			break
		}
	}

	// Save the commands to the file
	if err := saveUrlCommand(); err != nil {
		return c.String(http.StatusOK, "Failed to save your command, please try again later")
	}

	return c.String(http.StatusOK, "Updated Successfully")
}

func DeleteCommand(c echo.Context) error {
	command := c.FormValue("command")
	textName := c.FormValue("text")

	// Validate the command
	err := validateCommand(command)
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// Make sure command name does not exist
	urlNameExist := findUrlName(textName)

	if urlNameExist == nil {
		return c.String(http.StatusOK, fmt.Sprintf("Could not delete the command '%s' was not found.", textName))
	}

	// Delete the url command
	for i := range urlList {
		if urlList[i].Name == textName {
			urlList = append(urlList[:i], urlList[i+1:]...)
			break
		}
	}

	// Save the commands to the file
	if err := saveUrlCommand(); err != nil {
		return c.String(http.StatusOK, "Failed to save your command, please try again later")
	}

	return c.String(http.StatusOK, "Deleted Successfully")
}

func getNameAndUrl(c echo.Context) (string, string, error) {
	// Text should look like this: meet https://meet.google.com
	text := c.FormValue("text")

	// Split the text input into name and URL
	parts := strings.Fields(text) // This will split by whitespace

	if len(parts) < 2 {
		return "", "", fmt.Errorf("text must contain a name and a URL, ex: /add meet https://meet.google.com")
	}

	textName := parts[0]
	textURL := parts[1]

	return textName, textURL, nil
}

func findUrlName(name string) *UrlCommand {
	for _, command := range urlList {
		if command.Name == name {
			return &command
		}
	}
	return nil
}

// ValidateURL checks if the URL meets the criteria
func validateURL(url string) bool {
	// Check if URL has at least 6 characters
	if len(url) < 6 {
		return false
	}

	// Regular expression for URLs starting with http, https, or www
	re := regexp.MustCompile(`^(https?://|www\.)`)
	return re.MatchString(url)
}

func linkBlock(url, text string) map[string]interface{} {
	title := cases.Title(language.English)
	return map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": title.String(text),
		},
		"accessory": map[string]interface{}{
			"type": "button",
			"text": map[string]interface{}{
				"type":  "plain_text",
				"text":  "Go",
				"emoji": true,
			},
			"value":     "go-link",
			"url":       "https://echo.labstack.com/docs/middleware/cors#usage-1",
			"action_id": "button-action",
		},
	}
}
