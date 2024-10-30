package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/slack-go/slack"
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
		var blocks []interface{}
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("Here's a list of go commands in *%s*.", text),
			},
		})
		for _, cmd := range matchingUrlName {
			blocks = append(blocks, linkBlock(cmd.URL, cmd.Name))
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"blocks": blocks,
		})
	}
	// if no command is found
	return c.JSON(http.StatusOK, map[string]interface{}{
		"blocks": []interface{}{
			map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("No command found with the name '%s'. Please check the name and try again.", text),
				},
			},
			map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": "You can use the `/list` command to see all available commands.",
				},
			},
		},
	})
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
		return c.String(http.StatusOK, fmt.Sprintf("The command name '%s' was not found. Please make sure the name exists.", textName))
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

	return c.String(http.StatusOK, fmt.Sprintf("The command '%s' has been updated successfully with the new URL '%s'", textName, textUrl))
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

func ListCommand(c echo.Context) error {
	command := c.FormValue("command")

	// Validate the command
	err := validateCommand(command)
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	// List all the commands
	var commands = make(map[string][]interface{})

	for _, cmd := range urlList {
		nameParts := strings.Split(cmd.Name, "-")

		if len(nameParts) > 0 {
			group := nameParts[0]
			commands[group] = append(commands[group], linkBlock(cmd.URL, cmd.Name))
		}
	}

	responseBlocks := []interface{}{
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": "*Here's a list of all available commands:*",
			},
		},
		map[string]interface{}{
			"type": "divider",
		},
	}
	caser := cases.Title(language.English)
	for group, cmds := range commands {
		// Show the title
		responseBlocks = append(responseBlocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s*", caser.String(group)),
			},
		})

		// show the list of go commands
		responseBlocks = append(responseBlocks, cmds...)

		responseBlocks = append(responseBlocks, map[string]interface{}{
			"type": "divider",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"blocks": responseBlocks,
	})
}

func Interaction(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func OnEvent(c echo.Context) error {
	println("Event received")
	data := make(map[string]interface{})
	err := json.NewDecoder(c.Request().Body).Decode(&data)
	if err != nil {
		return c.String(http.StatusBadRequest, "Failed to parse request body")
	}

	eventType := data["type"].(string)

	if eventType == "url_verification" {
		challenge := data["challenge"].(string)
		return c.JSON(http.StatusOK, map[string]string{
			"challenge": challenge,
		})
	}
	return dispatchEvent(data, c)
}

func dispatchEvent(data map[string]interface{}, c echo.Context) error {
	eventType := data["event"].(map[string]interface{})["type"].(string)
	switch eventType {
	case "message":
		return handleMessageEvent(data, c)
	}

	return nil
}

func handleMessageEvent(data map[string]interface{}, c echo.Context) error {
	event := data["event"].(map[string]interface{})
	text := event["text"].(string)
	channelID := event["channel"].(string)
	triggerChar := "-&ast;"
	// Check if there is an arrow in the text
	if strings.Contains(text, triggerChar) {
		// Split by arrow character to get the part after the arrow
		parts := strings.Split(text, triggerChar)
		if len(parts) > 1 {
			// Split the second part by spaces and get the first word
			wordAfterArrow := strings.Fields(parts[1])[0]
			// Check if the word after the arrow is a command
			command := findUrlName(wordAfterArrow)

			if command != nil {
				sendCommandMessage(command, channelID)
			}
		} else {
			fmt.Println("No arrow found in text")
		}
	}

	return c.String(http.StatusOK, "ok")
}

func sendCommandMessage(command *UrlCommand, channelID string) {
	// Send a message to the user
	token := os.Getenv("SLACK_BOT_TOKEN")

	api := slack.New(token)

	// Send "hello world" message to the specified channel
	_, _, err := api.PostMessage(channelID, slack.MsgOptionBlocks(linkBlock(command.URL, command.Name)))
	if err != nil {
		fmt.Printf("error sending message: %v\n", err)
		return
	}
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
	// Must start with http, https
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}
	return true
}

func linkBlock(url, text string) slack.Block {
	title := cases.Title(language.English)
	formattedText := title.String(strings.ReplaceAll(text, "-", " "))
	return slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", formattedText, false, false),
		nil,
		slack.NewAccessory(
			slack.NewButtonBlockElement("", "", slack.NewTextBlockObject("plain_text", "⚡ Go", true, false)).WithStyle("primary").WithURL(url),
		),
	)
}
