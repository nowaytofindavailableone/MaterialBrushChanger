package modules

import (
	"encoding/json"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/sqweek/dialog"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Структура Config для хранения путей и названия проекта
type Config struct {
	FilePath    string `json:"file_path"`    // Путь до r4LavaEditor2.sessions.ini
	Workspace   string `json:"workspace"`    // Путь до рабочей директории (workspace)
	ProjectName string `json:"project_name"` // Название проекта
}

var configPath = "config.json"

// Функция GetPaths проверяет конфигурацию, если она недействительна, предлагает пользователю выбрать путь
func GetPaths() (string, string, string, error) {
	// Пытаемся загрузить конфигурацию
	config, err := LoadConfig()
	if err != nil || config == nil {
		// Если конфигурация не загружена, создаем новую
		config = &Config{}
	}

	// Проверяем и получаем путь к r4LavaEditor2.sessions.ini
	fullPath, err := getRedkitPath(config)
	if err != nil {
		return "", "", "", err
	}

	// Проверяем и получаем путь к рабочему пространству и название проекта
	workspacePath, projectName, err := getWorkspaceAndProjectName(config)
	if err != nil {
		return "", "", "", err
	}

	// Сохраняем обновленную конфигурацию
	err = SaveConfig(config)
	if err != nil {
		return "", "", "", fmt.Errorf("ошибка сохранения конфигурации: %v", err)
	}

	fmt.Printf("Paths saved to config.json: REDkit Path: %s, Workspace Path: %s, Project Name: %s\n", fullPath, workspacePath, projectName)
	return fullPath, workspacePath, projectName, nil
}

// Функция для загрузки конфигурации
func LoadConfig() (*Config, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Функция для сохранения конфигурации
func SaveConfig(config *Config) error {
	file, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configPath, file, 0644)
}

// Функция проверки наличия файла или директории
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getRedkitPath(config *Config) (string, error) {
	if config.FilePath != "" && FileExists(config.FilePath) {
		fmt.Printf("Last selected file exists: %s\n", config.FilePath)
		return config.FilePath, nil
	}

	fmt.Println("No valid configuration found for REDkit.")
	fmt.Println("Please select the folder path to The Witcher 3 REDkit or bin directory (bin\\r4LavaEditor2.sessions.ini).")

	// Открываем диалог для выбора директории
	folder, err := dialog.Directory().Title("Select The Witcher 3 REDkit or bin Folder").Browse()
	if err != nil {
		return "", fmt.Errorf("ошибка выбора директории: %v", err)
	}

	var fullPath string

	// Проверка, содержит ли путь подпапку "The Witcher 3 REDkit"
	if strings.Contains(strings.ToLower(folder), "the witcher 3 redkit") {
		// Если выбран путь к The Witcher 3 REDkit, добавляем bin и r4LavaEditor2.sessions.ini
		fullPath = filepath.Join(folder, "bin", "r4LavaEditor2.sessions.ini")
	} else if strings.HasSuffix(strings.ToLower(folder), "bin") {
		// Если пользователь выбрал директорию bin, используем ее напрямую
		fullPath = filepath.Join(folder, "r4LavaEditor2.sessions.ini")
	} else {
		// Если путь не содержит ни REDkit, ни bin, ищем "The Witcher 3 REDkit\bin" внутри
		redkitBinPath := filepath.Join(folder, "The Witcher 3 REDkit", "bin", "r4LavaEditor2.sessions.ini")
		if FileExists(redkitBinPath) {
			fullPath = redkitBinPath
		} else {
			// Если путь не содержит ни REDkit, ни bin, и не найден "The Witcher 3 REDkit\bin" внутри, возвращаем ошибку
			return "", fmt.Errorf("ошибка: выбранная директория не содержит ни 'The Witcher 3 REDkit', ни 'bin', ни 'The Witcher 3 REDkit\\bin'")
		}
	}

	// Проверяем существование файла
	if !FileExists(fullPath) {
		return "", fmt.Errorf("ошибка: файл r4LavaEditor2.sessions.ini не найден в выбранной папке")
	}

	// Сохраняем путь в конфигурацию
	config.FilePath = fullPath
	return fullPath, nil
}

func getWorkspaceAndProjectName(config *Config) (string, string, error) {
	if config.Workspace != "" && FileExists(config.Workspace) && config.ProjectName != "" {
		fmt.Printf("Last selected workspace and project exist: %s, %s\n", config.Workspace, config.ProjectName)
		return config.Workspace, config.ProjectName, nil // Возвращаем данные из конфигурации
	}

	fmt.Println("Please select the workspace folder (workspace).")
	workspaceFolder, err := dialog.Directory().Title("Select Workspace Folder").Browse()
	if err != nil {
		return "", "", fmt.Errorf("ошибка выбора директории workspace: %v", err)
	}

	// Проверяем, выбрана ли папка "workspace" или папка выше
	if filepath.Base(workspaceFolder) != "workspace" {
		// Если выбрана папка выше, ищем "workspace" внутри нее
		workspacePath := filepath.Join(workspaceFolder, "workspace")
		if !FileExists(workspacePath) {
			return "", "", fmt.Errorf("ошибка: папка 'workspace' не найдена внутри выбранной директории")
		} else {
			workspaceFolder = workspacePath
		}
	}

	config.Workspace = workspaceFolder

	dlcPath := filepath.Join(config.Workspace, "dlc")
	if !FileExists(dlcPath) {
		return "", "", fmt.Errorf("ошибка: папка 'dlc' не найдена в workspace")
	}

	projectName, err := getProjectNameWithFilter(dlcPath)
	if err != nil {
		return "", "", fmt.Errorf("ошибка получения названия проекта: %v", err)
	}

	// Закрытие консоли после завершения работы с ней
	if err := closeConsole(); err != nil {
		fmt.Printf("Ошибка при закрытии консоли: %v\n", err)
	}

	config.ProjectName = projectName
	return config.Workspace, projectName, nil
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procAllocConsole = kernel32.NewProc("AllocConsole")
var procFreeConsole = kernel32.NewProc("FreeConsole")

func closeConsole() error {
	r, _, err := procFreeConsole.Call()
	if r == 0 {
		return fmt.Errorf("не удалось закрыть консоль: %v", err)
	}
	return nil
}

func showConsole() (*os.File, error) {
	r, _, err := procAllocConsole.Call()
	if r == 0 {
		return nil, fmt.Errorf("не удалось открыть консоль: %v", err)
	}

	// Получаем дескриптор новой консоли
	consoleHandle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить дескриптор консоли: %v", err)
	}

	// Открываем файл, связанный с дескриптором консоли
	consoleFile := os.NewFile(uintptr(consoleHandle), "/dev/stdout")
	if consoleFile == nil {
		return nil, fmt.Errorf("не удалось открыть файл консоли")
	}

	// Перенаправляем stdout в новую консоль
	os.Stdout = consoleFile

	return consoleFile, nil
}

func getProjectNameWithFilter(dlcPath string) (string, error) {
	entries, err := os.ReadDir(dlcPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения папки 'dlc': %v", err)
	}

	var pafFolders []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "paf") {
			pafFolders = append(pafFolders, entry.Name())
		}
	}

	if len(pafFolders) == 0 {
		return "", fmt.Errorf("ошибка: папки, начинающиеся на 'paf', не найдены в 'dlc'")
	} else if len(pafFolders) == 1 {
		return pafFolders[0], nil
	} else if len(pafFolders) > 1 {
		// Открываем консольное окно для взаимодействия
		consoleFile, err := showConsole()
		if err != nil {
			return "", err
		}
		defer func(consoleFile *os.File) {
			err := consoleFile.Close()
			if err != nil {

			}
		}(consoleFile) // Закрываем файл консоли в конце функции

		// Если найдено несколько папок, используем консольный ввод для выбора
		fmt.Println("Найдено несколько проектов:")
		for i, folder := range pafFolders {
			fmt.Printf("%d. %s\n", i+1, folder)
		}

		for {
			input := prompt.Input("Выберите номер проекта: ", completer)
			choice, err := strconv.Atoi(input)
			if err != nil || choice < 1 || choice > len(pafFolders) {
				fmt.Println("Ошибка: неверный номер проекта")
				continue
			}

			return pafFolders[choice-1], nil
		}
	}

	return "", fmt.Errorf("неожиданная ошибка")
}

func completer(d prompt.Document) []prompt.Suggest {
	// Здесь можно добавить автодополнение, если нужно
	return []prompt.Suggest{}
}
