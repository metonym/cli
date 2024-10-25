package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	btable "github.com/evertras/bubble-table/table"
	"github.com/renderinc/render-cli/pkg/client"
	"github.com/renderinc/render-cli/pkg/command"
	"github.com/renderinc/render-cli/pkg/config"
	"github.com/renderinc/render-cli/pkg/owner"
	"github.com/renderinc/render-cli/pkg/tui"
	"github.com/spf13/cobra"
)

var workspaceSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Select a workspace to run commands against",
	Long: `Select a workspace to run commands against.
Your specified workspace will be saved in a config file specified by the RENDER_CLI_CONFIG_PATH environment variable.
If unspecified, the config file will be saved in $HOME/.render/cli.yaml. All subsequent commands will run against this workspace.

Currently, you can only select a workspace in interactive mode.`,
}

var InteractiveWorkspaceSet = command.Wrap(workspaceSetCmd, loadWorkspaceData, renderWorkspaces, nil)

type ListWorkspaceInput struct{}

const (
	columnWorkspaceIDKey    = "ID"
	columnWorkspaceNameKey  = "Name"
	columnWorkspaceEmailKey = "Email"
)

func loadWorkspaceData(ctx context.Context, _ ListWorkspaceInput) ([]*client.Owner, error) {
	c, err := client.NewDefaultClient()
	if err != nil {
		return nil, err
	}

	ownerRepo := owner.NewRepo(c)
	result, err := ownerRepo.ListOwners(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func renderWorkspaces(
	ctx context.Context,
	loadData func(input ListWorkspaceInput) tui.TypedCmd[[]*client.Owner],
	input ListWorkspaceInput,
) (tea.Model, error) {
	columns := []btable.Column{
		btable.NewColumn(columnWorkspaceIDKey, "ID", 28).WithFiltered(true),
		btable.NewFlexColumn(columnWorkspaceNameKey, "Name", 1).WithFiltered(true),
		btable.NewFlexColumn(columnWorkspaceEmailKey, "Email", 1).WithFiltered(true),
	}

	createRowFunc := func(owner *client.Owner) btable.Row {
		return btable.NewRow(btable.RowData{
			"ID":    owner.Id,
			"Name":  owner.Name,
			"Email": owner.Email,
		})
	}

	onSelect := func(rows []btable.Row) tea.Cmd {
		return func() tea.Msg {
			if len(rows) == 0 {
				return nil
			}

			selectedID, ok := rows[0].Data["ID"].(string)
			if !ok {
				return nil
			}

			owners, err := loadWorkspaceData(ctx, input)
			if err != nil {
				return tui.ErrorMsg{Err: fmt.Errorf("failed to load owners: %w", err)}
			}

			for _, o := range owners {
				if o.Id == selectedID {
					return selectWorkspace(o)
				}
			}

			return nil
		}
	}

	t := tui.NewTable(
		columns,
		loadData(input),
		createRowFunc,
		onSelect,
	)

	return t, nil
}

func selectWorkspace(o *client.Owner) tea.Msg {
	conf, err := config.Load()
	if err != nil {
		return tui.ErrorMsg{Err: fmt.Errorf("failed to load config: %w", err)}
	}

	conf.Workspace = o.Id
	if err := conf.Persist(); err != nil {
		return tui.ErrorMsg{Err: fmt.Errorf("failed to persist config: %w", err)}
	}

	return tui.DoneMsg{Message: fmt.Sprintf("Workspace set to %s", o.Name)}
}

func init() {
	workspaceSetCmd.RunE = func(cmd *cobra.Command, args []string) error {
		var input ListWorkspaceInput
		err := command.ParseCommand(cmd, args, &input)
		if err != nil {
			return err
		}
		InteractiveWorkspaceSet(cmd.Context(), input)
		return nil
	}

	workspaceCmd.AddCommand(workspaceSetCmd)
}