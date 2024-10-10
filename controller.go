package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"text/template"

	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
)

// BaseWatermark содержит основные параметры для водяного знака
type BaseWatermark struct {
	Opacity float64     // Прозрачность водяного знака
	Color   color.Color // Цвет водяного знака
	Font    string      // Шрифт для текстового водяного знака
	Size    float64     // Размер текста водяного знака
	Rotate  float64     // Угол вращения текста водяного знака
}

// Watermark представляет собой графический водяной знак
type Watermark struct {
	BaseWatermark         // Встраивание базовых параметров
	Path          string  // Путь к файлу изображения водяного знака
	Scale         float64 // Масштабирование водяного знака
}

// TextWatermark представляет собой текстовый водяной знак
type TextWatermark struct {
	BaseWatermark        // Встраивание базовых параметров
	Text          string // Текст для водяного знака
}

// CreateImage создает изображение с текстовым водяным знаком
func (t *TextWatermark) CreateImage(width, height float64) image.Image {
	dc := gg.NewContext(int(width), int(height)) // Создаем новый контекст рисования
	dc.SetRGBA(1, 1, 1, 0)                       // Устанавливаем прозрачный фон
	dc.Clear()                                   // Очищаем контекст
	dc.SetColor(t.Color)                         // Устанавливаем цвет текста
	if err := dc.LoadFontFace(t.Font, t.Size); err != nil {
		panic(err) // Обрабатываем ошибку при загрузке шрифта
	}
	angle := -t.Rotate * (3.14 / 180)                          // Преобразуем угол в радианы
	dc.Push()                                                  // Запоминаем текущее состояние контекста
	dc.RotateAbout(angle, width/2, height/2)                   // Поворачиваем контекст
	dc.DrawStringAnchored(t.Text, width/2, height/2, 0.5, 0.5) // Рисуем текст в центре
	dc.Pop()                                                   // Восстанавливаем состояние контекста
	return dc.Image()                                          // Возвращаем изображение
}

// ApplyToImage накладывает графический водяной знак на базовое изображение
func (w *Watermark) ApplyToImage(baseImage image.Image) image.Image {
	bounds := baseImage.Bounds()                                   // Получаем границы базового изображения
	result := image.NewRGBA(bounds)                                // Создаем новое изображение для результата
	draw.Draw(result, bounds, baseImage, image.Point{}, draw.Over) // Рисуем базовое изображение на результирующем

	if w.Path != "" { // Проверяем, задан ли путь к изображению водяного знака
		watermarkFile, err := os.Open(w.Path) // Открываем файл водяного знака
		if err != nil {
			panic(err) // Обрабатываем ошибку при открытии файла
		}
		defer watermarkFile.Close()                      // Закрываем файл после завершения работы
		watermarkImage, err := png.Decode(watermarkFile) // Декодируем изображение водяного знака
		if err != nil {
			panic(err) // Обрабатываем ошибку
		}

		if w.Scale != 1 { // Проверяем, требуется ли изменение масштаба
			bounds := watermarkImage.Bounds()                                                    // Получаем границы изображения водяного знака
			newWidth := uint(float64(bounds.Dx()) * w.Scale)                                     // Рассчитываем новый размер по ширине
			newHeight := uint(float64(bounds.Dy()) * w.Scale)                                    // Рассчитываем новый размер по высоте
			watermarkImage = resize.Resize(newWidth, newHeight, watermarkImage, resize.Bilinear) // Масштабируем изображение
		}

		// Рассчитываем позицию накладываемого водяного знака (центр)
		offset := image.Point{
			X: (bounds.Dx() - watermarkImage.Bounds().Dx()) / 2,
			Y: (bounds.Dy() - watermarkImage.Bounds().Dy()) / 2,
		}

		// Накладываем водяной знак
		draw.Draw(result, watermarkImage.Bounds().Add(offset), watermarkImage, image.Point{}, draw.Over)
	}

	return result // Возвращаем изображение с наложенным водяным знаком
}

// CreateWatermarkedImage создает изображение с текстовым водяным знаком
func (w *TextWatermark) CreateWatermarkedImage(baseImage image.Image) image.Image {
	textImage := w.CreateImage(float64(baseImage.Bounds().Dx()), float64(baseImage.Bounds().Dy())) // Генерируем текстовый водяной знак
	result := image.NewRGBA(baseImage.Bounds())                                                    // Создаем блок для результата
	draw.Draw(result, baseImage.Bounds(), baseImage, image.Point{}, draw.Over)                     // Копируем базовое изображение в результат

	// Накладываем текстовый водяной знак
	draw.DrawMask(result, result.Bounds(), textImage, image.Point{}, &image.Uniform{color.Alpha{uint8(255 * w.Opacity)}}, image.Point{}, draw.Over)
	return result // Возвращаем итоговое изображение
}

// encodeImageToBase64 кодирует изображение в формат Base64
func encodeImageToBase64(img image.Image, imgType string) string {
	buff := new(bytes.Buffer) // Создаем новый буфер для кодирования
	switch imgType {
	case "jpeg":
		jpeg.Encode(buff, img, nil) // Кодируем в формате JPEG
	case "png":
		png.Encode(buff, img) // Кодируем в формате PNG
	}
	return base64.StdEncoding.EncodeToString(buff.Bytes()) // Возвращаем строку Base64
}

// handleWatermarkedImages обрабатывает запрос для создания изображений с водяными знаками
func handleWatermarkedImages(w http.ResponseWriter, r *http.Request) {
	// Создаем экземпляр графического водяного знака
	imageWatermark := &Watermark{
		BaseWatermark: BaseWatermark{
			Opacity: 0.6, // Устанавливаем прозрачность
		},
		Path:  "FG-copyright-mini.png", // Путь к изображению водяного знака
		Scale: 1.0,                     // Масштабируем без изменений
	}

	// Загружаем первое изображение
	srcImgFile1, err := os.Open("image.jpg") // Открываем файл изображения
	if err != nil {
		panic(err) // Обрабатываем ошибку
	}
	defer srcImgFile1.Close()                       // Закрываем файл после работы
	baseImage1, _, err := image.Decode(srcImgFile1) // Декодируем изображение
	if err != nil {
		panic(err) // Обрабатываем ошибку
	}
	watermarkedImage1 := imageWatermark.ApplyToImage(baseImage1) // Применяем водяной знак

	// Создаем текстовый водяной знак
	textWatermark := &TextWatermark{
		BaseWatermark: BaseWatermark{
			Opacity: 0.6,                           // Прозрачность текста
			Color:   color.RGBA{239, 250, 23, 255}, // Цвет текста
			Font:    "Nunito-Medium.ttf",           // Путь к используемому шрифту
			Size:    35,                            // Размер текста
			Rotate:  -29.5,                         // Угол поворота текста
		},
		Text: "пятаяпередача.рф", // Текст водяного знака
	}

	// Загружаем второе изображение
	srcImgFile2, err := os.Open("zerkalo-ozera.jpg") // Открываем файл второго изображения
	if err != nil {
		panic(err) // Обрабатываем ошибку
	}
	defer srcImgFile2.Close()                       // Закрываем файл после работы
	baseImage2, _, err := image.Decode(srcImgFile2) // Декодируем второе изображение
	if err != nil {
		panic(err) // Обрабатываем ошибку
	}
	watermarkedImage2 := textWatermark.CreateWatermarkedImage(baseImage2) // Применяем текстовый водяной знак

	// Кодируем итоговые изображения в формат Base64
	imageWithWatermarkBase64_1 := encodeImageToBase64(watermarkedImage1, "jpeg")
	imageWithWatermarkBase64_2 := encodeImageToBase64(watermarkedImage2, "jpeg")

	// Используем шаблонизатор для отображения изображений
	tmpl, err := template.ParseFiles("templates/images.html") // Загружаем шаблон HTML
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError) // Обрабатываем ошибку загрузки шаблона
		return
	}

	// Подготовка данных для передачи в шаблон
	data := struct {
		Image1 string // Кодированное изображение с водяным знаком 1
		Image2 string // Кодированное изображение с водяным знаком 2
	}{
		Image1: imageWithWatermarkBase64_1, // Передаем первое изображение
		Image2: imageWithWatermarkBase64_2, // Передаем второе изображение
	}

	// Устанавливаем заголовок и выполняем шаблон
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError) // Обрабатываем ошибку при выполнении шаблона
	}
}
