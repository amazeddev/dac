package interact

import (
	"github.com/AlecAivazis/survey/v2"
)

// StringPrompt asks for a string value using the label
func StringPrompt(label string) string {
	result := ""
	prompt := &survey.Input{
			Message: label,
	}
	survey.AskOne(prompt, &result)
	return result
}

// YesNoPrompt asks yes/no questions using the label.
func ConfirmPrompt(label string) bool {
	result := false
	prompt := &survey.Confirm{
			Message: label,
	}
	survey.AskOne(prompt, &result)
	return result
}

func SelectPrompt(label string, options []string) string {
	result := ""
	prompt := &survey.Select{
		Message: label,
		Options: options,
	}
	survey.AskOne(prompt, &result)
	return result
}