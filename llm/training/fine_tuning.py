from transformers import (GPT2LMHeadModel, GPT2Tokenizer, Trainer, TrainingArguments,
                          AutoTokenizer, AutoModelForCausalLM, DataCollatorForLanguageModeling)
from datasets import load_dataset

# Загрузка модели и токенизатора
#model_name = "ai-forever/rugpt3small_based_on_gpt2"
model_name = "ai-forever/rugpt3medium_based_on_gpt2"
tokenizer = AutoTokenizer.from_pretrained(model_name)
model = AutoModelForCausalLM.from_pretrained(model_name)

# Подготовка данных
dataset = load_dataset("text", data_files={"train": "./split_datasets/trainv2.txt",
                                           "validation": "./split_datasets/valv2.txt"})

#print(dataset["train"][:5])


def tokenize_function(examples):
    tokenized = tokenizer(examples["text"],
                          truncation=True,
                          max_length=512,
                          padding="max_length",
                          return_tensors="pt")
    # Filter out empty sequences
    return tokenized
    #return {k: v for k, v in tokenized.items() if len(v) > 0}


tokenized_datasets = dataset.map(tokenize_function, batched=True)
tokenized_datasets = tokenized_datasets.filter(lambda x: len(x["input_ids"]) > 0)


data_collator = DataCollatorForLanguageModeling(
    tokenizer=tokenizer,
    mlm=False,  # mlm=False для моделей GPT (causal LM)
)

# Аргументы обучения
training_args = TrainingArguments(
    output_dir="./results",
    per_device_train_batch_size=4,
    num_train_epochs=3,
    save_steps=10_000,
    evaluation_strategy="epoch",
    logging_dir="./logs",
)

# Обучение
# В Trainer добавляем data_collator
trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=tokenized_datasets["train"],
    eval_dataset=tokenized_datasets["validation"],
    data_collator=data_collator,  # Автоматически генерирует labels
)

trainer.train()

model.save_pretrained("./finetuned_models/my_finetuned_model_rugpt3mediumv2")
tokenizer.save_pretrained("./finetuned_models/my_finetuned_model_rugpt3mediumv2")

input_text = """Ты являешься психологом, вежливым и сдержанным собеседником. Отвечаешь чётко и ясно.
 Пользователь: Подскажи, как распределить обязанности по дому между мной и моей девушкой?
 AI:"""
inputs = tokenizer(input_text, return_tensors="pt")
outputs = model.generate(**inputs, num_beams=5, max_length=100, early_stopping=True, no_repeat_ngram_size=2)
print(tokenizer.decode(outputs[0], skip_special_tokens=True))

test_dataset = load_dataset("text", data_files={"test": "./split_datasets/testv2.txt"})
tokenized_test = test_dataset.map(tokenize_function, batched=True)
trainer.evaluate(tokenized_test["test"])

# Загрузка для дальнейшего использования
# model = GPT2LMHeadModel.from_pretrained("./my_finetuned_model")