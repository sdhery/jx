package cmd

import (
	"io"

	"errors"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

type CreateGkeServiceAccountFlags struct {
	Name      string
	Project   string
	SkipLogin bool
}

type CreateGkeServiceAccountOptions struct {
	CreateOptions
	Flags CreateGkeServiceAccountFlags
}

var (
	createGkeServiceAccountExample = templates.Examples(`
		jx create gke-service-account

		# to specify the options via flags
		jx create gke-service-account --name my-service-account --project my-gke-project

`)
)

// NewCmdCreateGkeServiceAccount creates a command object for the "create" command
func NewCmdCreateGkeServiceAccount(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGkeServiceAccountOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "gke-service-account",
		Short:   "Creates a GKE service account",
		Example: createGkeServiceAccountExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd)

	return cmd
}

func (options *CreateGkeServiceAccountOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Flags.Name, "name", "n", "", "The name of the service account to create")
	cmd.Flags().StringVarP(&options.Flags.Project, "project", "p", "", "The GCP project to create the service account in")
	cmd.Flags().BoolVarP(&options.Flags.SkipLogin, "skip-login", "", false, "Skip Google auth if already logged in via gloud auth")
}

// Run implements this command
func (o *CreateGkeServiceAccountOptions) Run() error {
	if !o.Flags.SkipLogin {
		err := o.runCommandVerbose("gcloud", "auth", "login", "--brief")
		if err != nil {
			return err
		}
	}

	if o.Flags.Name == "" {
		prompt := &survey.Input{
			Message: "Name for the service account",
		}

		err := survey.AskOne(prompt, &o.Flags.Name, func(val interface{}) error {
			// since we are validating an Input, the assertion will always succeed
			if str, ok := val.(string); !ok || len(str) < 6 {
				return errors.New("Service Account name must be longer than 5 characters")
			}
			return nil
		})

		if err != nil {
			return err
		}

	}

	if o.Flags.Project == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		o.Flags.Project = projectId
	}

	path, err := gke.GetOrCreateServiceAccount(o.Flags.Name, o.Flags.Project, util.HomeDir())
	if err != nil {
		return err
	}

	log.Infof("Created service account key %s\n", util.ColorInfo(path))

	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (o *CreateGkeServiceAccountOptions) getGoogleProjectId() (string, error) {
	existingProjects, err := gke.GetGoogleProjects()
	if err != nil {
		return "", err
	}

	var projectId string
	if len(existingProjects) == 0 {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("No existing Google Projects exist, create one now?"),
			Default: true,
		}
		flag := true
		err = survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return "", err
		}
		if !flag {
			return "", errors.New("no google project to create cluster in, please manual create one and rerun this wizard")
		}

		if flag {
			return "", errors.New("auto creating projects not yet implemented, please manually create one and rerun the wizard")
		}
	} else if len(existingProjects) == 1 {
		projectId = existingProjects[0]
		log.Infof("Using the only Google Cloud Project %s to create the cluster\n", util.ColorInfo(projectId))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		err := survey.AskOne(prompts, &projectId, nil)
		if err != nil {
			return "", err
		}
	}

	if projectId == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}
