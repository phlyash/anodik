package integration

import "fmt"

const zedTasks = `
[
    {
        "label": "Build debug",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s build",
        "use_new_terminal": false,
    },
    {
        "label": "Build release",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s build --release",
        "use_new_terminal": false,
    },
    {
        "label": "Clean",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s clean",
        "use_new_terminal": false,
        "reveal": "always",
        "hide": "on_success",
    },
    {
        "label": "Run",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s run",
        "use_new_terminal": false,
    },
    {
        "label": "Start openocd",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s ocd",
        "use_new_terminal": false,
    },
    {
        "label": "Stop background openocd",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s ocd --stop",
        "use_new_terminal": false,
        "hide": "always",
    },

    {
        "label": "Flash",
        "allow_concurrent_runs": false,
        "cwd": "$ZED_WORKTREE_ROOT",
        "command": "%[1]s flash",
        "use_new_terminal": false,
        "reveal": "always",
        "hide": "on_success",
    },
]
`

const zedSettings = `
{
    "lsp": {
        "clangd": {
            "binary": { "path": "%[1]s" }
        }
    },
    "languages": {
        "C++": {
            "formatter": {
                "external": {
                    "command": "%[2]s",
                }
            }
        },
        "C": {
            "formatter": {
                "external": {
                    "command": "%[2]s",
                }
            }
        }
    }
}
`

func ZedTasks(anodikPath string) string {
	return fmt.Sprintf(zedTasks, anodikPath)
}

func ZedSettings(clangd string, clangFormat string) string {
	return fmt.Sprintf(zedSettings, clangd, clangFormat)
}
