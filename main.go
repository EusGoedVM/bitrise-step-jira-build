package main

import (
	"os"

	"github.com/Holdapp/bitrise-step-jira-build/bitrise"
	"github.com/Holdapp/bitrise-step-jira-build/config"
	"github.com/Holdapp/bitrise-step-jira-build/service"
	logger "github.com/bitrise-io/go-utils/log"
	"regexp"

	"github.com/bitrise-io/go-steputils/stepconf"
)

type StepConfig struct {
	// Generar info
	AppVersion string `env:"app_version,required"`

	// JIRA
	JiraHost         string          `env:"jira_host,required"`
	JiraUsername     string          `env:"jira_username,required"`
	JiraToken        stepconf.Secret `env:"jira_access_token,required"`
	JiraFieldID      int             `env:"jira_custom_field_id,required"`
	JiraURLFieldID	 int     				 `env:"jira_url_custom_field_id"`
	JiraIssuePattern string          `env:"jira_issue_pattern,required"`

	// Bitrise API
	BitriseToken stepconf.Secret `env:"bitrise_api_token,required"`

	// Fields provided by Bitrise
	BuildNumber string `env:"BITRISE_BUILD_NUMBER,required"`
	Workflow    string `env:"BITRISE_TRIGGERED_WORKFLOW_TITLE,required"`
	SourceDir   string `env:"BITRISE_SOURCE_DIR,required"`
	Branch      string `env:"BITRISE_GIT_BRANCH,required"`
	BuildSlug   string `env:"BITRISE_BUILD_SLUG,required"`
	AppSlug     string `env:"BITRISE_APP_SLUG,required"`
	InstallURL  string `env:"BITRISE_PUBLIC_INSTALL_PAGE_URL"`
}

func (config *StepConfig) JiraTokenString() string {
	return string(config.JiraToken)
}

func (config *StepConfig) BitriseTokenString() string {
	return string(config.BitriseToken)
}

func main() {
	// Parse config
	var stepConfig = StepConfig{}
	if err := stepconf.Parse(&stepConfig); err != nil {
		logger.Errorf("Configuration error: %s", err)
		os.Exit(1)
	}

	build := config.Build{
		Version: stepConfig.AppVersion,
		Number:  stepConfig.BuildNumber,
	}

	// get commit hashes from bitrise
	logger.Infof("Scanning Bitrise API for previous failed/aborted builds\n")
	bitriseClient := bitrise.Client{Token: stepConfig.BitriseTokenString()}
	_, err := service.ScanRelatedCommits(
		&bitriseClient, stepConfig.AppSlug,
		stepConfig.BuildSlug, stepConfig.Workflow,
		stepConfig.Branch,
	)

	if err != nil {
		logger.Errorf("Bitrise error: %s\n", err)
		os.Exit(2)
	}

	regex, err := regexp.Compile(stepConfig.JiraIssuePattern)
	if err != nil {
		logger.Errorf("Error in regex stuff: %v", err)
	}

	var issue = regex.FindAllString(stepConfig.Branch, -1)

	// update custom field on issues with current build number
	logger.Infof("Updating build status for issues: %v\n", issue)
	jiraWorker, err := service.NewJIRAWorker(
		stepConfig.JiraHost, stepConfig.JiraUsername,
		stepConfig.JiraTokenString(), stepConfig.JiraFieldID,
		stepConfig.JiraURLFieldID,
	)
	if err != nil {
		logger.Errorf("JIRA error: %s\n", err)
		os.Exit(4)
	}

	jiraWorker.UpdateBuildForIssues(issue, build, stepConfig.InstallURL)
	logger.Infof("Updated ticket %s with build number %s", issue[0], stepConfig.BuildNumber)
	// exit with success code
	os.Exit(0)
}
