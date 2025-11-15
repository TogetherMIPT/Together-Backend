from transformers import AutoTokenizer, AutoModelForCausalLM, GPT2LMHeadModel, TextStreamer

#

# tokenizer = AutoTokenizer.from_pretrained("ai-forever/rugpt3medium_based_on_gpt2")
# model = AutoModelForCausalLM.from_pretrained("ai-forever/rugpt3medium_based_on_gpt2")
# tokenizer = AutoTokenizer.from_pretrained("ai-forever/rugpt3small_based_on_gpt2")
# model = AutoModelForCausalLM.from_pretrained("ai-forever/rugpt3small_based_on_gpt2")

# Загрузка для дальнейшего использования
model = AutoModelForCausalLM.from_pretrained("./finetuned_models/my_finetuned_model_rugpt3smallv2")
tokenizer = AutoTokenizer.from_pretrained("./finetuned_models/my_finetuned_model_rugpt3smallv2")
streamer = TextStreamer(tokenizer, skip_prompt=True, skip_special_tokens=True, clean_up_tokenization_spaces=True)

#input_text = "Привет! Как дела?"
input_text = """Ты являешься психологом, вежливым и сдержанным собеседником. Отвечаешь чётко и ясно.
 Пользователь: Подскажи, как распределить обязанности по дому между мной и моей девушкой?
 AI:"""

input_text = """Пользователь 1: «Я просто устал от всего. Мы постоянно ссоримся из-за мелочей, например, кто должен выносить мусор. Кажется, она совсем меня не уважает».
Пользователь 2: «Это не про мусор! Речь о том, что я чувствую, будто все бытовые вопросы тяну на себе, а он этого даже не замечает».
Как ответить этой паре?"""

inputs = tokenizer(input_text, return_tensors="pt")
# outputs = model.generate(**inputs, num_beams=5, max_length=200, early_stopping=True, no_repeat_ngram_size=2)
# print(tokenizer.decode(outputs[0]))
model.generate(**inputs, max_new_tokens=100, no_repeat_ngram_size=2,
               streamer=streamer, temperature=0.7, do_sample=True)




