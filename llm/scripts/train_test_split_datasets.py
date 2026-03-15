import os
from sklearn.model_selection import train_test_split


def unite_texts(source_dir):
    # Считываем все текстовые файлы
    texts = []
    for filename in os.listdir(source_dir):
        if filename.endswith(".txt"):
            with open(os.path.join(source_dir, filename), "r", encoding="utf-8", errors="replace") as f:
                texts.append(f.read())

    print(f"Всего текстов: {len(texts)}")
    return texts


def save_to_file(texts, output_dir, filename):
    os.makedirs(output_dir, exist_ok=True)
    with open(os.path.join(output_dir, filename), "w", encoding="utf-8") as f:
        for text in texts:
            f.write(text.strip() + "\n\n")  # Добавляем пустую строку между документами


if __name__ == '__main__':
    texts = unite_texts("../dataset/prep_dataset/")
    # Первое разделение: 80% train, 20% временный набор
    train_texts, temp_texts = train_test_split(texts, test_size=0.2, random_state=42)

    # Второе разделение: 50% val, 50% test (от temp_texts)
    val_texts, test_texts = train_test_split(temp_texts, test_size=0.5, random_state=42)

    print(f"Train: {len(train_texts)}, Val: {len(val_texts)}, Test: {len(test_texts)}, Test+Val: {len(temp_texts)}")

    save_to_file(train_texts,  "../dataset/split_datasets/", "train.txt")
    save_to_file(val_texts,  "../dataset/split_datasets/", "val.txt")
    save_to_file(test_texts,  "../dataset/split_datasets/", "test.txt")
    save_to_file(temp_texts, "../dataset/split_datasets/", "testandval.txt")
