package modules

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Структура для хранения блока пресета
type PresetBlock struct {
	Header string
	Values map[string]string
}

// Массив с заданным порядком ключей
var keysOrder = []string{
	"VerticalMask",
	"VerticalUVMult",
	"VerticalUVScaleMask",
	"HeightLowLimit",
	"Probability",
	"PresetEnabled",
	"HighLimitMask",
	"HorizontalMask",
	"SelectedHorizontalTexture",
	"SlopeThresholdMask",
	"SelectedVerticalTexture",
	"LowLimitMask",
	"SlopeThresholdAction",
	"SlopeThresholdIndex",
	"HeightHighLimit",
}

// Функция для замены блоков в .ini файле на блоки из пресета
func ReplaceBlocksInIni(sessionFilePath, presetFilePath, projectName string) {
	// Считываем блоки из пресета
	presetBlocks := ParsePresetBlocks(presetFilePath)

	// Открываем файл сессии для чтения с системной кодировкой
	sessionFile, err := os.Open(sessionFilePath)
	if err != nil {
		fmt.Printf("Error opening session file: %v\n", err)
		return
	}

	// Открываем временный файл для записи (замена блоков будет происходить здесь)
	tempFilePath := sessionFilePath + ".tmp"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		fmt.Printf("Error creating temp file: %v\n", err)
		sessionFile.Close()
		return
	}

	// Оборачиваем ридеры и райтеры для работы с системной кодировкой
	reader := transform.NewReader(sessionFile, charmap.Windows1251.NewDecoder())
	writer := transform.NewWriter(tempFile, charmap.Windows1251.NewEncoder())

	// Сканируем файл построчно
	scanner := bufio.NewScanner(reader)
	inBlock := false
	var currentBlock PresetBlock

	for scanner.Scan() {
		line := scanner.Text()

		// Извлекаем номер слота из заголовка блока
		re := regexp.MustCompile(`MaterialPairSlot(\d+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			slotNumber := matches[1]

			// Формируем ожидаемый заголовок блока для текущего слота
			expectedBlockHeader := fmt.Sprintf("[Session/dlc\\%s\\data\\levels\\%s\\%s.w2w/Tools/TerrainEdit/MaterialPairSlot%s]", projectName, projectName, projectName, slotNumber)

			// Определяем начало нового блока, сравнивая с ожидаемым заголовком
			if line == expectedBlockHeader {
				if inBlock {
					// Проверяем, есть ли этот блок в пресете
					if newBlock, found := presetBlocks[currentBlock.Header]; found {
						// Если блок найден, заменяем его содержимым из пресета
						writeBlockToFile(newBlock, writer)
						fmt.Printf("Block replaced: %s\n", currentBlock.Header)
					} else {
						// Если блок не найден, записываем старый блок как есть
						writeBlockToFile(currentBlock, writer)
					}
				}

				// Начинаем новый блок
				currentBlock = PresetBlock{
					Header: line,
					Values: make(map[string]string),
				}
				inBlock = true
			} else if inBlock {
				// Парсим ключ-значение в блоке
				parts := strings.Split(line, "=")
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					currentBlock.Values[key] = value

					// Если достигли конца блока (ключ "HeightHighLimit"), блок завершен
					if key == "HeightHighLimit" {
						if newBlock, found := presetBlocks[currentBlock.Header]; found {
							// Если блок найден в пресете, заменяем его содержимым из пресета
							writeBlockToFile(newBlock, writer)
							fmt.Printf("Block replaced: %s\n", currentBlock.Header)
						} else {
							// Если блок не найден, записываем старый блок как есть
							writeBlockToFile(currentBlock, writer)
						}
						inBlock = false
					}
				}
			} else {
				// Записываем строки, не относящиеся к блокам, напрямую в новый файл
				writer.Write([]byte(line + "\r\n")) // Добавляем \r\n для DOS-стиля
			}
		}
	}

	// Проверка на ошибки сканирования
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading session file: %v\n", err)
		sessionFile.Close()
		tempFile.Close()
		return
	}

	// Закрываем оригинальный файл после завершения чтения
	sessionFile.Close()

	// Если файл завершился на блоке, проверяем последний блок
	if inBlock {
		if newBlock, found := presetBlocks[currentBlock.Header]; found {
			// Если блок найден в пресете, заменяем его содержимым из пресета
			writeBlockToFile(newBlock, writer)
			fmt.Printf("Block replaced: %s\n", currentBlock.Header)
		} else {
			// Если блок не найден, записываем старый блок как есть
			writeBlockToFile(currentBlock, writer)
		}
	}

	// Закрываем временный файл
	tempFile.Close()

	// Заменяем оригинальный файл на обновленный
	err = os.Remove(sessionFilePath)
	if err != nil {
		fmt.Printf("Error removing original file: %v\n", err)
		return
	}

	err = os.Rename(tempFilePath, sessionFilePath)
	if err != nil {
		fmt.Printf("Error renaming temp file: %v\n", err)
		return
	}

	fmt.Println("Replacement completed successfully.")
}

// Функция для считывания блоков из файла пресетов
func ParsePresetBlocks(filePath string) map[string]PresetBlock {
	blocks := make(map[string]PresetBlock)

	// Открываем файл с системной кодировкой
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening preset file: %v\n", err)
		return blocks
	}
	defer file.Close()

	reader := transform.NewReader(file, charmap.Windows1251.NewDecoder())

	// Читаем файл построчно
	scanner := bufio.NewScanner(reader)
	var currentBlock PresetBlock
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		// Если строка начинается с '[', то это новый блок
		if strings.HasPrefix(line, "[") {
			if inBlock {
				// Завершаем текущий блок и сохраняем его
				blocks[currentBlock.Header] = currentBlock
			}
			// Начинаем новый блок
			currentBlock = PresetBlock{
				Header: line,
				Values: make(map[string]string),
			}
			inBlock = true
		} else if inBlock {
			// Парсим ключ-значение в блоке
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentBlock.Values[key] = value
			}
		}
	}

	// Добавляем последний блок
	if inBlock {
		blocks[currentBlock.Header] = currentBlock
	}

	fmt.Printf("Parsed %d blocks from preset file.\n", len(blocks))
	return blocks
}

// Функция для записи блока в файл с DOS-стилем перевода строк
func writeBlockToFile(block PresetBlock, writer io.Writer) {
	writer.Write([]byte(block.Header + "\r\n")) // DOS-стиль перевода строки \r\n
	for i, key := range keysOrder {
		if value, exists := block.Values[key]; exists {
			if i == len(keysOrder)-1 {
				writer.Write([]byte(fmt.Sprintf("%s=%s\r\n", key, value))) // Последняя строка тоже с \r\n
			} else {
				writer.Write([]byte(fmt.Sprintf("%s=%s\r\n", key, value)))
			}
		}
	}
}
