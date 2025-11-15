import pdfplumber
import pypandoc
import cv2
import pytesseract
import easyocr
from pdf2image import convert_from_path
from PIL import Image
import re
import os


def remove_toc(text):
    """
    Удаление содержания
    :param text:
    :return:
    """
    # Удаляем строки вида "Текст ............... 123"
    pattern = r'^.*?\.{3,}\s*\d+\s*$'
    # Разбиваем текст на строки и фильтруем
    # lines = [line for line in text.split('\n') if not re.fullmatch(pattern, line.replace(' ', ''))]
    lines = text.split('\n')
    lines_cleaned = []
    for i in range(len(lines)):
        if not re.fullmatch(pattern, lines[i].replace(' ', '')) \
                and (i + 1 == len(lines) or not re.fullmatch(pattern, lines[i + 1].replace(' ', ''))):
            lines_cleaned.append(lines[i])
    # Дополнительно удаляем строку "Оглавление", если она есть
    lines = [line for line in lines_cleaned if not line.strip().lower() == 'оглавление'
             or line.strip().lower() == 'содержание']
    return '\n'.join(lines)


def fix_hyphenated_words(text):
    """
    Склеиваем слова, разорванные дефисом и переносом строки
    :param text:
    :return:
    """
    # В случае неверного распознавания символа "-" символом "G" делается замена
    text = re.sub(r'(\w+)-\s*\n\s*(\w+)', r'\1\2', text.replace('G', '-'))

    # text = re.sub(r'(\w+)G\s*\n\s*(\w+)', r'\1\2', text)
    return text


def remove_latin_digits_only_lines(text):
    """
    Удаляет строки, содержащие только:
    - латинские буквы (a-z, A-Z)
    - цифры (0-9)
    - знаки препинания (.,!?;:)
    - скобки (()[]{})
    - технические символы (#№$%&@*+-=/\)
    - пробельные символы
    :param text:
    :return:
    """
    pattern = r'^[\w\s.,!?;:\'"«»„“”‘’‒–—―()\[\]{}#№$%&@*+=<>\/\\^_`|~-]+$'
    # Разбиваем текст на строки и фильтруем
    lines = [line for line in text.split('\n')
             if not re.fullmatch(pattern, line.strip())]

    return '\n'.join(lines)


def clean_text(text):
    """
    Очистка текста
    :param text:
    :return:
    """
    # Склеиваем слова, разорванные дефисом и переносом строки
    text = fix_hyphenated_words(text)
    # Удаление номеров страниц
    lines = [line for line in text.split('\n') if not line.replace('.', '').strip().isdigit()]
    text = ' \n'.join(lines)
    # Удаление номеров страниц (с подписями страница)
    text = re.sub(r'Page \d+|Страница \d+', '', text)
    # Удаление оглавления
    text = remove_toc(text)
    text = re.sub(r'Алфавитный указатель.*', '', text, flags=re.DOTALL)
    # Удаляем строки с указанием глав и частей
    chapter_pattern = r'^(Глава|Часть|Раздел|Chapter|Part|Section)\s*[IVXLCDM0-9]+[.:]?.*$'
    text = re.sub(chapter_pattern, '', text, flags=re.MULTILINE | re.IGNORECASE)
    # Удаление переносов строк
    # text = text.replace('-\n', '')
    # Удаление лишних пробелов и переносов
    text = ' '.join(text.split())
    # Удаление колонититулов
    text = re.sub(r'^(.*?)\n(.*?)\n', '', text)
    # Удаление URL-ссылок
    text = re.sub(r'http[s]?://\S+|www\.\S+', '', text)
    # Удаление ссылок в квадратных скобках [1], [пример]
    text = re.sub(r'\[[^\]]+\]', '', text)
    # Удаление специальных символов
    text = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', text)
    text = text.replace('♦', '').replace('■', '')
    # Удаление cid-кодов (спец. символы из PDF)
    text = re.sub(r'\(cid:\d+\)', '', text)
    # Удаление списка литературы
    # text = remove_latin_digits_only_lines(text) # может удалить все строки из книги???
    pattern = r'(?i)(?:список\s+литературы|references|bibliography)[\s\S]*?$'
    text = re.sub(pattern, '', text).strip()
    # Нормализация Unicode
    text = text.encode('utf-8', 'ignore').decode('utf-8')
    return text


def extract_text_from_file(source_path, txt_path, clean_flag=0, pdf_image=0):
    """
    Получение текста из файлов разного формата
    :param source_path:
    :param txt_path:
    :param clean_flag:
    :param pdf_image:
    :return:
    """
    with open(txt_path, "w", encoding='utf-8') as target_file:
        if source_path.lower().endswith('.pdf'):
            if pdf_image != 1:
                with pdfplumber.open(source_path) as pdf:
                    i = 0
                    for page in pdf.pages:
                        print("page: ", i)
                        if pdf_image == 0:
                            text = page.extract_text()
                        elif pdf_image == 2:
                            temp_img_path = "./images/temp.png"
                            im = page.to_image()  # конвертируем страницу в изображение
                            im.save(temp_img_path)  # сохраняем временное изображение
                            reader = easyocr.Reader(['ru', 'en'])
                            text = reader.readtext(temp_img_path, detail=0)
                            text = '\n'.join(text)
                            os.remove(temp_img_path)
                        if clean_flag:
                            text = clean_text(text)
                        target_file.write(text)
                        i += 1
            else:
                # #print("YES")
                # temp_img_path = "./images/temp.png"
                # im = page.to_image()  # конвертируем страницу в изображение
                # im.save(temp_img_path)  # сохраняем временное изображение
                # image = cv2.imread(temp_img_path)
                # gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
                # thresh = cv2.threshold(gray, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)[1]
                # text = pytesseract.image_to_string(thresh, lang="rus")  # распознаём текст
                images = convert_from_path(source_path, dpi=300)
                for i, img in enumerate(images):
                    print("page: ", i)
                    temp_img_path = f"./images/page_{i}.jpg"
                    img.save(temp_img_path, "JPEG")
                    text = pytesseract.image_to_string(
                        temp_img_path,
                        lang='rus+eng',
                        config='--psm 6 --oem 3'  # Режим для блоков текста
                    )
                    if clean_flag:
                        text = clean_text(text)
                    target_file.write(text)
                    os.remove(temp_img_path)
        elif source_path.lower().endswith('.fb2'):
            tmp_res_file = txt_path + '_tmp'
            pypandoc.convert_file(
                source_path,
                'plain',
                outputfile=tmp_res_file
            )
            with open(tmp_res_file, 'r', encoding='utf-8') as f:
                text = f.read()
                text = clean_text(text)
                target_file.write(text)
            os.remove(tmp_res_file)

        elif source_path.lower().endswith('.txt'):
            with open(source_path, 'r', encoding='utf-8') as f:
                text = f.read()
                text = clean_text(text)
                target_file.write(text)


def make_dataset(source_directory, target_directory, clean_flag=0, pdf_image=0):
    """
    Подготовка датасета для обучения LLM-модели
    :param source_directory:
    :param target_directory:
    :param clean_flag:
    :param pdf_image:
    :return:
    """
    for root, _, files in os.walk(source_directory):
        for filename in files:
            source_path = os.path.join(root, filename)
            #txt_path = os.path.join(target_directory, filename[:-4] + '_cleaned.txt')
            txt_path = os.path.join(target_directory, filename[:-4] + '.txt')
            print(source_path)
            extract_text_from_file(source_path, txt_path, clean_flag, pdf_image)


if __name__ == '__main__':
    # extract_text_from_pdf('./books/Karen_Khorni_Zhenskaya_psikhologia.pdf',
    #                       './txt_files/Karen_Khorni_Zhenskaya_psikhologia_cleaned.txt', 1)
    # extract_text_from_pdf('books/Kniga-Pol-i-gender-Ilin-E.pdf',
    #                        './txt_files/Kniga-Pol-i-gender-Ilin-E_cleaned.txt', 1)
    make_dataset('./books_test', './dataset2', clean_flag=1, pdf_image=1)
