# 使用方法
1. 转录（Transcription）
将音频转换为原语言文本

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://api.qingyuntop.top/v1",
    api_key=key
)
```

# 基础转录
```python
audio_file = open("/path/to/file/audio.mp3", "rb")
transcription = client.audio.transcriptions.create(
  model="whisper-1",
  file=audio_file
)
print(transcription.text)
# 指定输出格式
transcription = client.audio.transcriptions.create(
  model="whisper-1",
  file=audio_file,
  response_format="text"
)
```
2. 翻译（Translation）

将任意语言音频转换为英文文本
```python
from openai import OpenAI

client = OpenAI(
    base_url="https://api.qingyuntop.top/v1",
    api_key=key
)

audio_file = open("/path/to/file/german.mp3", "rb")
translation = client.audio.translations.create(
  model="whisper-1",
  file=audio_file
)
print(translation.text)
```

3. 时间戳功能
```python
from openai import OpenAI

client = OpenAI(
    base_url="https://api.qingyuntop.top/v1",
    api_key=key
)

audio_file = open("speech.mp3", "rb")
transcript = client.audio.transcriptions.create(
  file=audio_file,
  model="whisper-1",
  response_format="verbose_json",
  timestamp_granularities=["word"]
)

print(transcript.words)
```


4. 处理大文件
使用 PyDub 分割大于25MB的文件：
```python
from pydub import AudioSegment

song = AudioSegment.from_mp3("good_morning.mp3")

# 分割为10分钟片段
ten_minutes = 10 * 60 * 1000
first_10_minutes = song[:ten_minutes]
first_10_minutes.export("good_morning_10.mp3", format="mp3")
```