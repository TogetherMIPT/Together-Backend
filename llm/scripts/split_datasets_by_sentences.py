import os
import re
from typing import List, Tuple

try:
    import nltk
    from nltk.tokenize import sent_tokenize
    nltk.data.find('tokenizers/punkt')
except (ImportError, LookupError):
    nltk = None
    print("NLTK не найден или не загружен. Устанавливаю данные...")
    try:
        nltk.download('punkt', quiet=True)
        from nltk.tokenize import sent_tokenize
    except:
        sent_tokenize = None


def robust_sent_tokenize(text: str) -> List[str]:
    """Надёжная токенизация предложений с резервным вариантом"""
    if sent_tokenize is not None:
        return sent_tokenize(text)
    
    # Резервный метод: разделение по знакам препинания с пробелом/концом строки
    sentences = re.split(r'(?<=[.!?])\s+', text.strip())
    return [s.strip() for s in sentences if s.strip()]


def extract_sentences_with_metadata(source_dir: str) -> List[Tuple[str, int, int]]:
    """
    Извлекает предложения с метаданными (исходный файл, номер документа, номер предложения)
    Возвращает список: [(предложение, doc_id, sent_id_in_doc), ...]
    """
    sentences = []
    doc_id = 0
    
    for filename in sorted(os.listdir(source_dir)):
        if not filename.endswith(".txt"):
            continue
            
        filepath = os.path.join(source_dir, filename)
        try:
            with open(filepath, "r", encoding="utf-8", errors="replace") as f:
                text = f.read()
            
            doc_sentences = robust_sent_tokenize(text)
            if not doc_sentences:
                continue
                
            # Добавляем разделитель между документами для сохранения контекста
            sentences.append(("<|doc_start|>", doc_id, -1))
            
            for sent_id, sent in enumerate(doc_sentences):
                if len(sent.strip()) > 5:  # фильтр слишком коротких фрагментов
                    sentences.append((sent.strip(), doc_id, sent_id))
            
            doc_id += 1
            
        except Exception as e:
            print(f"Ошибка обработки {filename}: {e}")
    
    print(f"Всего документов: {doc_id}")
    print(f"Всего предложений: {len([s for s in sentences if s[2] != -1])}")
    return sentences


def split_by_sentence_count(sentences: List[Tuple[str, int, int]], 
                           train_ratio=0.8, 
                           val_ratio=0.1) -> Tuple[List[str], List[str], List[str]]:
    """
    Разделение корпуса по количеству предложений в заданных пропорциях.
    Сохраняет разделители документов (<|doc_start|>).
    """
    # Фильтруем только настоящие предложения для расчёта пропорций
    real_sentences = [s for s in sentences if s[2] != -1]
    total = len(real_sentences)
    
    train_end = int(total * train_ratio)
    val_end = train_end + int(total * val_ratio)
    
    print(f"Предложений: всего={total}, train={train_end}, val={val_end-train_end}, test={total-val_end}")
    
    # Формируем выборки, сохраняя разделители документов
    train, val, test = [], [], []
    current_split = 'train'
    real_count = 0
    
    for sent, doc_id, sent_id in sentences:
        if sent_id == -1:  # разделитель документа
            if real_count < train_end:
                train.append(sent)
            elif real_count < val_end:
                val.append(sent)
            else:
                test.append(sent)
            continue
        
        if real_count < train_end:
            train.append(sent)
        elif real_count < val_end:
            val.append(sent)
        else:
            test.append(sent)
        
        real_count += 1
    
    return train, val, test


def save_sentences(sentences: List[str], output_dir: str, filename: str):
    """Сохраняет предложения в файл с двойным переводом строки между ними"""
    os.makedirs(output_dir, exist_ok=True)
    filepath = os.path.join(output_dir, filename)
    
    with open(filepath, "w", encoding="utf-8") as f:
        for sent in sentences:
            if sent == "<|doc_start|>":
                f.write("\n\n### НОВЫЙ ДОКУМЕНТ ###\n\n")
            else:
                f.write(sent + "\n\n")
    
    print(f"Сохранено {len([s for s in sentences if s != '<|doc_start|>'])} предложений в {filepath}")


def main():
    source_dir = "../dataset/prep_dataset/"
    output_dir = "../dataset/split_datasets/"
    
    # 1. Извлечение предложений
    sentences = extract_sentences_with_metadata(source_dir)
    
    if not sentences:
        raise ValueError("Не найдено ни одного предложения для обработки!")
    
    # 2. Разделение по количеству предложений (80/10/10)
    train_sents, val_sents, test_sents = split_by_sentence_count(
        sentences, 
        train_ratio=0.8, 
        val_ratio=0.1
    )
    
    # 3. Сохранение результатов
    save_sentences(train_sents, output_dir, "trainv3.txt")
    save_sentences(val_sents, output_dir, "valv3.txt")
    save_sentences(test_sents, output_dir, "testv3.txt")
    
    # Опционально: объединённый тест+валидация
    save_sentences(val_sents + test_sents, output_dir, "testandvalv3.txt")
    
    # Статистика
    total_real = len([s for s in sentences if s[2] != -1])
    train_real = len([s for s in train_sents if s != "<|doc_start|>"])
    val_real = len([s for s in val_sents if s != "<|doc_start|>"])
    test_real = len([s for s in test_sents if s != "<|doc_start|>"])
    
    print("\nИтоговая статистика:")
    print(f"Train:  {train_real:6d} ({train_real/total_real*100:5.2f}%)")
    print(f"Val:    {val_real:6d} ({val_real/total_real*100:5.2f}%)")
    print(f"Test:   {test_real:6d} ({test_real/total_real*100:5.2f}%)")
    print(f"Total:  {total_real:6d} (100.00%)")


if __name__ == '__main__':
    main()
    save_to_file(temp_texts, "../dataset/split_datasets/", "testandvalv3.txt")
