package integration

import "fmt"

const vscodeTasks = `
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Build debug",
            "type": "shell",
            "command": "%[1]s build",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared"
            },
            "runOptions": {
                "instanceLimit": 1
            }
        },
        {
            "label": "Build release",
            "type": "shell",
            "command": "%[1]s build --release",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared"
            },
            "runOptions": {
                "instanceLimit": 1
            }
        },
        {
            "label": "Clean",
            "type": "shell",
            "command": "%[1]s clean",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared",
                "close": true
            },
            "runOptions": {
                "instanceLimit": 1
            }
        },
        {
            "label": "Run",
            "type": "shell",
            "command": "%[1]s run",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared"
            },
            "runOptions": {
                "instanceLimit": 1
            }
        },
        {
            "label": "Start openocd",
            "type": "shell",
            "command": "%[1]s ocd",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared"
            },
            "runOptions": {
                "instanceLimit": 1
            },
            "isBackground": true
        },
        {
            "label": "Stop background openocd",
            "type": "shell",
            "command": "%[1]s ocd --stop",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "never",
                "panel": "shared"
            },
            "runOptions": {
                "instanceLimit": 1
            }
        },
        {
            "label": "Flash",
            "type": "shell",
            "command": "%[1]s flash",
            "options": {
                "cwd": "${workspaceFolder}"
            },
            "presentation": {
                "reveal": "always",
                "panel": "shared",
                "close": true
            },
            "runOptions": {
                "instanceLimit": 1
            }
        }
    ]
}
`

func VscodeTasks(anodikPath string) string {
	return fmt.Sprintf(vscodeTasks, anodikPath)
}
