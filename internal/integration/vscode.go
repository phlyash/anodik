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
            "label": "Start openocd in background",
            "type": "shell",
            "command": "%[1]s ocd --bg",
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

const vscodeDebug = `
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug with GDB (OpenOCD remote)",
            "type": "cppdbg",
            "request": "launch",
            "program": "${workspaceFolder}/build/%[1]s",
            "miDebuggerPath": "%[2]s",
            "cwd": "${workspaceFolder}",
            "MIMode": "gdb",
            "setupCommands": [
                { "text": "file $%[3]s/build/%[1]s" },
                { "text": "tar rem :3333" },
                { "text": "monitor reset halt" },
                { "text": "load" }
            ],
            "stopAtEntry": true,
        }
    ]
}
`

const vscodeSettings = `
{
    "clangd.path": "%s",
    "clang-format.executable": "%s",
}
`

func VscodeTasks(anodikPath string) string {
	return fmt.Sprintf(vscodeTasks, anodikPath)
}

func VscodeLaunch(projectName string, gdbPath string, projectDir string) string {
	return fmt.Sprintf(vscodeDebug, projectName, gdbPath, projectDir)
}

func VscodeSettings(clangd string, clangFormat string) string {
	return fmt.Sprintf(vscodeSettings, clangd, clangFormat)
}
