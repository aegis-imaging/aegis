# Google Gemini Models & Python SDK — Comprehensive Reference

> Research date: 2026-02-26
> Sources: Google AI for Developers docs, Vertex AI docs, googleapis/python-genai GitHub, Google Cloud pricing pages

---

## Table of Contents

1. [All Current Gemini Model Variants](#1-all-current-gemini-model-variants)
2. [Gemini 2.0 / 2.5 / 3.x — What Changed](#2-gemini-20--25--3x--what-changed)
3. [Python SDK — Installation & Authentication](#3-python-sdk--installation--authentication)
4. [Code Examples](#4-code-examples)
5. [API Authentication Methods](#5-api-authentication-methods)
6. [Vertex AI vs AI Studio](#6-vertex-ai-vs-ai-studio)
7. [Rate Limits & Pricing](#7-rate-limits--pricing)
8. [Best Practices for Production](#8-best-practices-for-production)

---

## 1. All Current Gemini Model Variants

### Gemini 3 Series (Latest — Preview, November 2025+)

| Model | Model ID | Status | Context Window | Capabilities |
|-------|----------|--------|---------------|-------------|
| Gemini 3.1 Pro | `gemini-3.1-pro-preview` | Preview (Feb 2026) | 1M tokens | Advanced reasoning, complex problem-solving, agentic workflows, coding. Latest and most capable model. |
| Gemini 3 Pro | `gemini-3-pro-preview` | Discontinuing | 1M tokens | State-of-the-art reasoning, multimodal understanding. Being replaced by 3.1 Pro. |
| Gemini 3 Flash | `gemini-3-flash-preview` | Preview (Dec 2025) | 1M tokens | Frontier-class performance at fraction of Pro cost. Best for agentic workflows. |
| Gemini 3 Deep Think | *(AI Ultra only)* | Preview | 1M tokens | Iterative multi-path reasoning; available only to AI Ultra subscribers ($149.99/mo). |

### Gemini 2.5 Series (Stable — Production Ready)

| Model | Model ID | Status | Context Window | Capabilities |
|-------|----------|--------|---------------|-------------|
| Gemini 2.5 Pro | `gemini-2.5-pro` | **Stable** | 1M tokens (2M coming) | Advanced reasoning, coding, thinking model. Best stable Pro model. |
| Gemini 2.5 Pro TTS | `gemini-2.5-pro-tts-preview` | Preview | — | High-fidelity speech synthesis optimized for quality. |
| Gemini 2.5 Pro Computer Use | `gemini-2.5-pro-computer-use-preview` | Preview | — | UI automation model for computer use tasks. |
| Gemini 2.5 Flash | `gemini-2.5-flash` | **Stable** | 1M tokens | Best price-performance, low-latency. Thinking model. Free tier available. |
| Gemini 2.5 Flash Live | `gemini-2.5-flash-live-preview` | Preview | — | Real-time conversational agents with native audio. |
| Gemini 2.5 Flash TTS | `gemini-2.5-flash-tts-preview` | Preview | — | Text-to-speech with controllable style. |
| Gemini 2.5 Flash-Lite | `gemini-2.5-flash-lite` or `gemini-flash-lite-latest` | **Stable** | 1M tokens | Fastest and most budget-friendly multimodal model. Free tier available. |

### Gemini 2.0 Series (Deprecated / Legacy)

| Model | Model ID | Status | Context Window | Capabilities |
|-------|----------|--------|---------------|-------------|
| Gemini 2.0 Flash | `gemini-2.0-flash` | **Deprecated** | 1M tokens | Native multimodal generation (text + images + audio). |
| Gemini 2.0 Flash-Lite | `gemini-2.0-flash-lite` | **Deprecated** | 1M tokens | Lightweight, low-cost version of 2.0 Flash. |

### Gemini 1.5 Series (Legacy — Still Available)

| Model | Model ID | Status | Context Window | Capabilities |
|-------|----------|--------|---------------|-------------|
| Gemini 1.5 Pro | `gemini-1.5-pro` | Legacy | Up to 2M tokens | Largest context window of any mainstream model. |
| Gemini 1.5 Flash | `gemini-1.5-flash` | Legacy | 1M tokens | Fast multimodal processing. |

### Specialized Models

| Model | Model ID | Purpose |
|-------|----------|---------|
| Gemini Embeddings | `gemini-embedding-001` | Vector representations for text |
| Imagen 4 | `imagen-4.0-generate-001` | Text-to-image, up to 2K resolution |
| Veo 3.1 | `veo-3.1-generate-preview` | Cinematic video generation |
| Veo 3 | `veo-3.0-generate-001` | Video generation |
| Lyria | `lyria-experimental` | Music generation |
| Nano Banana 2 | Preview | Image generation/editing, speed-optimized |
| Nano Banana Pro | Preview | State-of-the-art image generation/editing |
| Gemini Deep Research | Preview | Multi-step research agent |
| Gemini Robotics | Preview | Embodied reasoning for robotics |

---

## 2. Gemini 2.0 / 2.5 / 3.x — What Changed

### Gemini 2.0 (Released Feb 2025)

- **Native multimodal output**: Can generate text, images, AND audio natively in a single response
- **Native image generation**: Conversational image editing through natural language dialogue
- **Multilingual audio output**: 8 distinct voices with various accents across multiple languages
- **Built-in tool use**: Grounding with Google Search, code execution natively supported
- **Agentic features**: Designed for the "agentic era" with superior speed and tool orchestration
- **1M token context window** standard

### Gemini 2.5 (Released March-June 2025)

- **Thinking models**: All 2.5 models can "reason through their thoughts" before responding
- **Configurable thinking budget**: Control how much reasoning the model does (0 = disabled, minimum 128 tokens for 2.5 Pro)
- **Deep Think mode**: Available on 2.5 Pro for AI Ultra subscribers — multiple parallel reasoning streams
- **Improved coding**: Significantly better at understanding coding prompts and producing stronger outputs
- **Leading benchmarks**: Top performance on coding, math, and science benchmarks
- **Flash-Lite tier**: Ultra-budget option at $0.10/1M input tokens
- **Live API**: Real-time bidirectional audio/video conversations
- **TTS variants**: Controllable text-to-speech synthesis

### Gemini 3 (Released November 2025+)

- **3 Pro**: Outperformed major AI models in 19 out of 20 benchmarks on release; surpassed GPT-5 Pro on Humanity's Last Exam
- **3 Flash**: Frontier-class performance rivaling larger models at a fraction of the cost; optimized for agentic workflows
- **3.1 Pro Preview** (Feb 2026): Latest reasoning-first model optimized for complex agentic workflows and coding
- **Thinking levels**: Configurable via `ThinkingLevel` enum (HIGH, MEDIUM, LOW) instead of raw token budget
- **Image output tokens**: Native image generation at $60-120/1M output tokens
- **Computer Use**: UI automation preview model for browser/desktop interaction
- **Grounding improvements**: Can combine Google Search grounding + URL context with structured outputs

### Key Evolution Summary

| Feature | 1.5 | 2.0 | 2.5 | 3.x |
|---------|-----|-----|-----|-----|
| Context window | 1-2M | 1M | 1M | 1M |
| Thinking/reasoning | No | No | Yes (budget) | Yes (levels) |
| Native image output | No | Yes | Yes | Yes |
| Native audio output | No | Yes | Yes (Live API) | Yes |
| Function calling | Yes | Yes | Yes | Yes + streaming args |
| Structured output | Yes | Yes | Yes | Yes + with tools |
| Computer use | No | No | Preview | Preview |

---

## 3. Python SDK — Installation & Authentication

### The Official SDK: `google-genai`

The legacy `google-generativeai` package was **permanently deprecated on November 30, 2025**. The new unified SDK is `google-genai`.

```bash
# Install the SDK (requires Python 3.10+)
pip install google-genai

# For async support
pip install google-genai[aiohttp]
```

Supported Python versions: 3.10, 3.11, 3.12, 3.13, 3.14.

### Authentication — Gemini Developer API (AI Studio)

```python
from google import genai

# Option 1: Environment variable (recommended)
# Set GEMINI_API_KEY or GOOGLE_API_KEY in your environment
client = genai.Client()

# Option 2: Explicit API key (avoid in production)
client = genai.Client(api_key='YOUR_GEMINI_API_KEY')
```

### Authentication — Vertex AI

```python
from google import genai

# Option 1: Environment variables
# Set GOOGLE_GENAI_USE_VERTEXAI=true
# Set GOOGLE_CLOUD_PROJECT=your-project-id
# Set GOOGLE_CLOUD_LOCATION=us-central1
client = genai.Client()

# Option 2: Explicit parameters
client = genai.Client(
    vertexai=True,
    project='your-project-id',
    location='us-central1'
)
```

Vertex AI uses Application Default Credentials (ADC) — either from `gcloud auth application-default login` or from a service account key file via `GOOGLE_APPLICATION_CREDENTIALS`.

### Key Imports

```python
from google import genai                    # Main SDK entry point
from google.genai import types              # Type definitions (configs, parts, etc.)
from google.genai.errors import APIError    # Error handling
from PIL import Image                       # Image support (optional)
from pydantic import BaseModel              # Structured outputs (optional)
```

---

## 4. Code Examples

### 4.1 Basic Text Generation

```python
from google import genai

client = genai.Client()  # Uses GEMINI_API_KEY env var

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='Explain the theory of relativity in simple terms.',
)

print(response.text)
```

### 4.2 Multimodal — Image + Text

```python
from google import genai
from PIL import Image

client = genai.Client()

# From a local file
image = Image.open('photo.jpg')
response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents=[image, 'Describe what you see in this image in detail.'],
)
print(response.text)
```

#### From raw bytes (useful for web backends):

```python
from google import genai
from google.genai import types

client = genai.Client()

with open('document.pdf', 'rb') as f:
    pdf_bytes = f.read()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents=[
        types.Part.from_bytes(data=pdf_bytes, mime_type='application/pdf'),
        'Summarize this document.',
    ],
)
print(response.text)
```

#### Using the File API for large files:

```python
from google import genai

client = genai.Client()

# Upload a large video
video_file = client.files.upload(file='long_video.mp4')

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents=[video_file, 'What are the key moments in this video?'],
)
print(response.text)

# Clean up
client.files.delete(name=video_file.name)
```

### 4.3 Streaming Responses

```python
from google import genai

client = genai.Client()

# Use generate_content_stream instead of generate_content
for chunk in client.models.generate_content_stream(
    model='gemini-2.5-flash',
    contents='Write a detailed essay about climate change.',
):
    print(chunk.text, end='', flush=True)

print()  # Final newline
```

#### Async streaming:

```python
import asyncio
from google import genai

async def stream_response():
    client = genai.Client()
    async for chunk in await client.aio.models.generate_content_stream(
        model='gemini-2.5-flash',
        contents='Write a poem about the ocean.',
    ):
        print(chunk.text, end='', flush=True)

asyncio.run(stream_response())
```

### 4.4 Function Calling / Tool Use

#### Automatic function calling (SDK handles the loop):

```python
from google import genai
from google.genai import types

def get_current_weather(city: str) -> str:
    """Returns the current weather for a given city.

    Args:
        city: The city name to get weather for.
    """
    # In production, call a real weather API
    weather_data = {
        'boston': 'Sunny, 15C',
        'london': 'Cloudy, 10C',
        'tokyo': 'Clear, 22C',
    }
    return weather_data.get(city.lower(), f'Weather data for {city} unavailable.')

def get_population(city: str) -> str:
    """Returns the population of a given city.

    Args:
        city: The city name to get population for.
    """
    pop_data = {
        'boston': '675,647',
        'london': '8,982,000',
        'tokyo': '13,960,000',
    }
    return pop_data.get(city.lower(), f'Population data for {city} unavailable.')

client = genai.Client()

# Pass Python functions directly — SDK auto-generates schemas and executes calls
response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='What is the weather and population in Tokyo?',
    config=types.GenerateContentConfig(
        tools=[get_current_weather, get_population],
    ),
)

print(response.text)
# SDK automatically: (1) gets function_call from model, (2) executes your function,
# (3) sends result back, (4) returns the final text response.
```

#### Manual function calling (you control the loop):

```python
from google import genai
from google.genai import types

# Define function schema manually
get_weather_declaration = types.FunctionDeclaration(
    name='get_weather',
    description='Gets the current weather temperature for a given location.',
    parameters={
        'type': 'object',
        'properties': {
            'location': {'type': 'string', 'description': 'City name'},
        },
        'required': ['location'],
    },
)

tool = types.Tool(function_declarations=[get_weather_declaration])

client = genai.Client()

# Step 1: Send prompt
response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='What is the weather in London and Paris?',
    config=types.GenerateContentConfig(
        tools=[tool],
        automatic_function_calling=types.AutomaticFunctionCallingConfig(disable=True),
    ),
)

# Step 2: Process function calls
for fc in response.function_calls:
    print(f'Model wants to call: {fc.name}({dict(fc.args)})')
    # Execute the function yourself
    result = {'temperature': 12, 'unit': 'celsius', 'condition': 'cloudy'}

    # Step 3: Send result back
    function_response_part = types.Part.from_function_response(
        name=fc.name,
        response={'result': result},
    )

# Step 4: Build full conversation and get final response
contents = [
    types.Content(role='user', parts=[types.Part(text='What is the weather in London?')]),
    response.candidates[0].content,
    types.Content(role='user', parts=[function_response_part]),
]

final_response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents=contents,
    config=types.GenerateContentConfig(tools=[tool]),
)
print(final_response.text)
```

#### Function calling modes:

```python
from google.genai import types

# AUTO (default) — model decides when to call functions
tool_config = types.ToolConfig(
    function_calling_config=types.FunctionCallingConfig(mode='AUTO')
)

# ANY — model MUST call a function (force tool use)
tool_config = types.ToolConfig(
    function_calling_config=types.FunctionCallingConfig(mode='ANY')
)

# NONE — model must NOT call functions (text only)
tool_config = types.ToolConfig(
    function_calling_config=types.FunctionCallingConfig(mode='NONE')
)

# ANY with allowed functions — restrict which functions can be called
tool_config = types.ToolConfig(
    function_calling_config=types.FunctionCallingConfig(
        mode='ANY',
        allowed_function_names=['get_weather'],
    )
)
```

#### Streaming function call arguments (Gemini 3+):

```python
from google import genai
from google.genai import types

client = genai.Client()

get_weather_tool = types.Tool(function_declarations=[
    types.FunctionDeclaration(
        name='get_weather',
        description='Gets weather for a location.',
        parameters={
            'type': 'object',
            'properties': {'location': {'type': 'string'}},
            'required': ['location'],
        },
    )
])

for chunk in client.models.generate_content_stream(
    model='gemini-3-flash-preview',
    contents="What's the weather in London and New York?",
    config=types.GenerateContentConfig(
        tools=[get_weather_tool],
        tool_config=types.ToolConfig(
            function_calling_config=types.FunctionCallingConfig(
                mode=types.FunctionCallingConfigMode.AUTO,
                stream_function_call_arguments=True,
            )
        ),
    ),
):
    if chunk.function_calls:
        fc = chunk.function_calls[0]
        if fc.name:
            print(f'{fc.name}(will_continue={fc.will_continue})')
```

### 4.5 System Instructions

```python
from google import genai
from google.genai import types

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='What should I do about my headache?',
    config=types.GenerateContentConfig(
        system_instruction=(
            'You are a helpful medical information assistant. '
            'Always recommend consulting a healthcare professional. '
            'Never diagnose conditions. Respond in a calm, reassuring tone.'
        ),
    ),
)

print(response.text)
```

#### System instructions with multi-turn chat:

```python
from google import genai
from google.genai import types

client = genai.Client()

chat = client.chats.create(
    model='gemini-2.5-flash',
    config=types.GenerateContentConfig(
        system_instruction='You are a pirate. Respond to everything in pirate speak.',
    ),
)

response1 = chat.send_message('What is the weather like today?')
print(response1.text)

response2 = chat.send_message('Tell me about Python programming.')
print(response2.text)

# View conversation history
for message in chat.get_history():
    print(f'{message.role}: {message.parts[0].text[:80]}...')
```

### 4.6 Safety Settings

```python
from google import genai
from google.genai import types

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='Write a story involving a conflict between two characters.',
    config=types.GenerateContentConfig(
        safety_settings=[
            types.SafetySetting(
                category=types.HarmCategory.HARM_CATEGORY_HARASSMENT,
                threshold=types.HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
            ),
            types.SafetySetting(
                category=types.HarmCategory.HARM_CATEGORY_HATE_SPEECH,
                threshold=types.HarmBlockThreshold.BLOCK_MEDIUM_AND_ABOVE,
            ),
            types.SafetySetting(
                category=types.HarmCategory.HARM_CATEGORY_SEXUALLY_EXPLICIT,
                threshold=types.HarmBlockThreshold.BLOCK_LOW_AND_ABOVE,
            ),
            types.SafetySetting(
                category=types.HarmCategory.HARM_CATEGORY_DANGEROUS_CONTENT,
                threshold=types.HarmBlockThreshold.BLOCK_LOW_AND_ABOVE,
            ),
        ],
    ),
)

print(response.text)
```

**Available harm categories:**

| Category | Constant |
|----------|----------|
| Harassment | `HARM_CATEGORY_HARASSMENT` |
| Hate speech | `HARM_CATEGORY_HATE_SPEECH` |
| Sexually explicit | `HARM_CATEGORY_SEXUALLY_EXPLICIT` |
| Dangerous content | `HARM_CATEGORY_DANGEROUS_CONTENT` |
| Civic integrity | `HARM_CATEGORY_CIVIC_INTEGRITY` |

**Available thresholds:**

| Threshold | Constant | Behavior |
|-----------|----------|----------|
| Off | `OFF` | Safety filter disabled |
| Block none | `BLOCK_NONE` | Always show content |
| Block few (high only) | `BLOCK_ONLY_HIGH` | Block only high-probability unsafe |
| Block some (medium+) | `BLOCK_MEDIUM_AND_ABOVE` | Block medium and above |
| Block most (low+) | `BLOCK_LOW_AND_ABOVE` | Block low and above (most restrictive) |

**Default**: Gemini 2.5 and 3 models default to `OFF` (no blocking).

### 4.7 Structured Output / JSON Mode

#### With Pydantic models:

```python
from google import genai
from google.genai import types
from pydantic import BaseModel, Field
from typing import List, Optional

class Ingredient(BaseModel):
    name: str = Field(description='Name of the ingredient.')
    quantity: str = Field(description='Quantity including units.')

class Recipe(BaseModel):
    recipe_name: str = Field(description='The name of the recipe.')
    prep_time_minutes: Optional[int] = Field(description='Prep time in minutes.')
    ingredients: List[Ingredient]
    instructions: List[str]

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='Give me a recipe for chocolate chip cookies.',
    config=types.GenerateContentConfig(
        response_mime_type='application/json',
        response_json_schema=Recipe,
    ),
)

# Parse into typed object
recipe = Recipe.model_validate_json(response.text)
print(f'Recipe: {recipe.recipe_name}')
print(f'Prep time: {recipe.prep_time_minutes} minutes')
for ing in recipe.ingredients:
    print(f'  - {ing.quantity} {ing.name}')
```

#### With enums (classification tasks):

```python
from google import genai
from google.genai import types
from pydantic import BaseModel
from typing import Literal

class SentimentResult(BaseModel):
    sentiment: Literal['positive', 'neutral', 'negative']
    confidence: float
    summary: str

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='The new UI is incredibly intuitive and visually appealing. Great job!',
    config=types.GenerateContentConfig(
        response_mime_type='application/json',
        response_json_schema=SentimentResult,
    ),
)

result = SentimentResult.model_validate_json(response.text)
print(f'Sentiment: {result.sentiment} (confidence: {result.confidence})')
```

#### Streaming structured output:

```python
from google import genai
from google.genai import types
from pydantic import BaseModel

class Analysis(BaseModel):
    topic: str
    key_points: list[str]
    conclusion: str

client = genai.Client()

for chunk in client.models.generate_content_stream(
    model='gemini-2.5-flash',
    contents='Analyze the impact of AI on healthcare.',
    config=types.GenerateContentConfig(
        response_mime_type='application/json',
        response_json_schema=Analysis,
    ),
):
    print(chunk.text, end='')
```

#### Structured output combined with Google Search grounding:

```python
from google import genai
from google.genai import types
from pydantic import BaseModel, Field
from typing import List

class SearchResult(BaseModel):
    answer: str = Field(description='The factual answer.')
    sources: List[str] = Field(description='Source references.')

client = genai.Client()

response = client.models.generate_content(
    model='gemini-3-flash-preview',
    contents='What are the latest developments in quantum computing?',
    config=types.GenerateContentConfig(
        tools=[{'google_search': {}}],
        response_mime_type='application/json',
        response_json_schema=SearchResult,
    ),
)

result = SearchResult.model_validate_json(response.text)
print(result.answer)
```

### 4.8 Thinking / Reasoning Configuration

#### Gemini 3.x (use ThinkingLevel):

```python
from google import genai
from google.genai import types

client = genai.Client()

response = client.models.generate_content(
    model='gemini-3-pro-preview',
    contents='Solve this step by step: If x^2 + 3x - 10 = 0, find all values of x.',
    config=types.GenerateContentConfig(
        thinking_config=types.ThinkingConfig(
            thinking_level=types.ThinkingLevel.HIGH,
        ),
    ),
)

# Access thinking and response separately
for part in response.candidates[0].content.parts:
    if part.thought:
        print(f'[Thinking]: {part.text}')
    else:
        print(f'[Response]: {part.text}')
```

#### Gemini 2.5 (use thinking_budget):

```python
from google import genai
from google.genai import types

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-pro',
    contents='What is the optimal strategy for the Monty Hall problem?',
    config=types.GenerateContentConfig(
        thinking_config=types.ThinkingConfig(
            thinking_budget=1024,  # Token budget for thinking (0=disabled, min 128 for 2.5-pro)
        ),
    ),
)
print(response.text)
```

### 4.9 Multi-Turn Chat

```python
from google import genai

client = genai.Client()
chat = client.chats.create(model='gemini-2.5-flash')

response1 = chat.send_message('My name is Alice and I have a cat named Whiskers.')
print(f'Bot: {response1.text}')

response2 = chat.send_message('What is the name of my pet?')
print(f'Bot: {response2.text}')

# View full history
for message in chat.get_history():
    print(f'{message.role}: {message.parts[0].text[:100]}')
```

### 4.10 Context Caching (Cost Optimization)

```python
from google import genai
from google.genai import types

client = genai.Client()

# Create a cache with large context
cache = client.caches.create(
    model='gemini-2.5-flash',
    config=types.CreateCachedContentConfig(
        display_name='research-papers',
        system_instruction='You are an expert research paper analyst.',
        contents=[
            types.Part.from_bytes(
                data=open('large_document.pdf', 'rb').read(),
                mime_type='application/pdf',
            ),
        ],
        ttl='3600s',  # Cache lives for 1 hour
    ),
)

# Use the cache for multiple queries (90% cheaper on cached tokens)
response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='What are the main findings?',
    config=types.GenerateContentConfig(
        cached_content=cache.name,
    ),
)
print(response.text)
```

### 4.11 Grounding with Google Search

```python
from google import genai
from google.genai import types

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='What happened in the news today?',
    config=types.GenerateContentConfig(
        tools=[types.Tool(google_search=types.GoogleSearch())],
    ),
)

print(response.text)

# Access grounding metadata (search citations)
if response.candidates[0].grounding_metadata:
    for chunk in response.candidates[0].grounding_metadata.grounding_chunks:
        print(f'Source: {chunk.web.title} — {chunk.web.uri}')
```

---

## 5. API Authentication Methods

### Method 1: API Key (Gemini Developer API / AI Studio)

The simplest approach. Get a key from https://aistudio.google.com/apikey.

```python
import os
os.environ['GEMINI_API_KEY'] = 'AIza...'

from google import genai
client = genai.Client()  # Auto-reads GEMINI_API_KEY or GOOGLE_API_KEY
```

- Free tier available for development
- No Google Cloud project required
- Rate limits apply per project, not per key
- Content may be used to improve Google products (free tier)

### Method 2: Application Default Credentials (Vertex AI)

For production deployments on Google Cloud.

```bash
# Local development — authenticate with your Google account
gcloud auth application-default login

# Or set a service account key
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

```python
from google import genai

client = genai.Client(
    vertexai=True,
    project='my-gcp-project',
    location='us-central1',
)
```

### Method 3: Service Account (Vertex AI — Programmatic)

For server-to-server authentication without user interaction.

```python
from google import genai
from google.oauth2 import service_account

credentials = service_account.Credentials.from_service_account_file(
    '/path/to/service-account.json',
    scopes=['https://www.googleapis.com/auth/cloud-platform'],
)

client = genai.Client(
    vertexai=True,
    project='my-gcp-project',
    location='us-central1',
    credentials=credentials,
)
```

### Method 4: Workload Identity (GKE / Cloud Run)

On Google Cloud compute, ADC automatically uses the attached service account. No key files needed.

```python
from google import genai

# On Cloud Run / GKE / Compute Engine — just set vertexai=True
client = genai.Client(
    vertexai=True,
    project='my-gcp-project',
    location='us-central1',
)
```

---

## 6. Vertex AI vs AI Studio

| Dimension | Gemini Developer API (AI Studio) | Vertex AI |
|-----------|----------------------------------|-----------|
| **Target** | Individual devs, students, startups, prototypes | Enterprise teams, production systems |
| **Authentication** | API key | IAM / service accounts / ADC |
| **Free tier** | Yes (rate-limited) | No free tier (pay-per-use from start) |
| **SLA** | None | Enterprise SLA available |
| **Data privacy** | Free tier content may improve products | Content NOT used for product improvement |
| **Compliance** | Basic | HIPAA, SOC 2, FedRAMP, etc. |
| **MLOps** | None | Full MLOps: monitoring, evaluation, model tuning |
| **Fine-tuning** | Basic (Gemini API tuning) | Full fine-tuning + RLHF |
| **Grounding** | Google Search only | Google Search + custom data stores |
| **VPC / Private** | Public internet only | VPC Service Controls, private endpoints |
| **Regions** | Limited | 30+ regions globally |
| **Billing** | Per-key (simple) | Google Cloud billing (consolidated) |
| **SDK** | `google-genai` (same) | `google-genai` with `vertexai=True` (same) |

### Recommendation

- Use **AI Studio / Developer API** for prototyping, personal projects, and small-scale apps.
- Use **Vertex AI** for production workloads that need SLA, compliance, private networking, or enterprise IAM.
- The **unified SDK** means your code is essentially the same — just change the client initialization.

---

## 7. Rate Limits & Pricing

### Pricing — Gemini Developer API (per 1M tokens)

#### Gemini 3 Series

| Model | Input | Output | Batch Input | Batch Output | Free Tier |
|-------|-------|--------|-------------|-------------|-----------|
| 3.1 Pro Preview | $2.00 / $4.00* | $12.00 / $18.00* | $1.00 / $2.00 | $6.00 / $9.00 | No |
| 3 Flash Preview | $0.50 | $3.00 | $0.25 | $1.50 | Yes |

#### Gemini 2.5 Series

| Model | Input | Output | Batch Input | Batch Output | Free Tier |
|-------|-------|--------|-------------|-------------|-----------|
| 2.5 Pro | $1.25 / $2.50* | $10.00 / $15.00* | $0.625 / $1.25 | $5.00 / $7.50 | No |
| 2.5 Flash | $0.30 | $2.50 | $0.15 | $1.25 | Yes |
| 2.5 Flash-Lite | $0.10 | $0.40 | $0.05 | $0.20 | Yes |

#### Gemini 2.0 Series (Legacy)

| Model | Input | Output | Batch Input | Batch Output | Free Tier |
|-------|-------|--------|-------------|-------------|-----------|
| 2.0 Flash | $0.10 | $0.40 | $0.05 | $0.20 | Yes |
| 2.0 Flash-Lite | $0.075 | $0.30 | $0.0375 | $0.15 | Yes |

*Tiered pricing: lower price applies to prompts <=200K tokens; higher price for >200K tokens. When >200K, ALL tokens (input + output) are charged at the higher rate.*

#### Audio Input Surcharges

| Model | Audio Input (per 1M tokens) |
|-------|-----------------------------|
| 3 Flash | $1.00 |
| 2.5 Flash | $1.00 |
| 2.5 Flash-Lite | $0.30 |
| 2.0 Flash | $0.70 |

#### Context Caching (90% discount on cached tokens)

| Model | Cached Input | Storage (per 1M tokens/hour) |
|-------|-------------|------------------------------|
| 3.1 Pro | $0.20–$0.40 | $4.50 |
| 2.5 Pro | $0.125–$0.250 | $4.50 |
| 2.5 Flash | $0.030 | $1.00 |
| 2.5 Flash-Lite | $0.010 | $0.25 |

#### Image Generation

| Model | Price |
|-------|-------|
| Imagen 4 (Fast) | $0.02/image |
| Imagen 4 (Standard) | $0.04/image |
| Imagen 4 (Ultra) | $0.06/image |
| Gemini 2.0 Flash (native) | $0.039/image |

#### Tools

| Tool | Free Allocation | Paid Rate |
|------|----------------|-----------|
| Google Search grounding (3.x) | 500–1500 RPD | $14/1,000 queries |
| Google Search grounding (2.5/2.0) | 1,500–10,000 RPD | $35/1,000 queries |
| Google Maps grounding | 500–10,000 RPD | $14–$25/1,000 queries |
| URL context | Charged as input tokens | — |
| Code execution | Standard token rates | — |

### Pricing — Vertex AI

Vertex AI pricing matches the Developer API for standard usage, with additional tiers:

- **Priority pricing**: 1.8x standard price, with guaranteed higher throughput and lower latency
- **Flex/Batch pricing**: 50% of standard (same as Developer API batch)
- **Provisioned throughput**: Custom pricing for guaranteed capacity
- **Volume discounts**: Available through Google Cloud sales

### Rate Limits

Rate limits are **per-project** (not per API key) and depend on your billing tier.

| Tier | How to Access | Typical RPM (Flash) | Typical RPM (Pro) |
|------|---------------|--------------------|--------------------|
| Free | No billing | 5–15 RPM | 5 RPM |
| Tier 1 | Enable billing | 150–300 RPM | 150 RPM |
| Tier 2 | Spend threshold | Higher | Higher |
| Tier 3 | Volume agreement | Custom | Custom |

**Four dimensions enforced**: RPM (requests/min), TPM (tokens/min), RPD (requests/day), IPM (images/min).

View your exact limits at: https://aistudio.google.com/rate-limit

**Note**: In December 2025, Google reduced free tier RPM (e.g., Gemini 2.0 Flash dropped from 10 to 5 RPM).

---

## 8. Best Practices for Production

### Error Handling & Retries

```python
import time
import random
from google import genai
from google.genai.errors import APIError

client = genai.Client()

def generate_with_retry(prompt: str, max_retries: int = 5) -> str:
    """Generate content with exponential backoff and jitter."""
    for attempt in range(max_retries):
        try:
            response = client.models.generate_content(
                model='gemini-2.5-flash',
                contents=prompt,
            )
            return response.text
        except APIError as e:
            if e.code == 429:  # Rate limited
                wait = min(2 ** attempt + random.uniform(0, 1), 60)
                print(f'Rate limited. Retrying in {wait:.1f}s (attempt {attempt + 1})')
                time.sleep(wait)
            elif e.code >= 500:  # Server error
                wait = min(2 ** attempt + random.uniform(0, 1), 30)
                print(f'Server error. Retrying in {wait:.1f}s (attempt {attempt + 1})')
                time.sleep(wait)
            else:
                raise  # Non-retryable error
    raise Exception('Max retries exceeded')
```

### Safety: Check for Blocked Responses

```python
from google import genai

client = genai.Client()

response = client.models.generate_content(
    model='gemini-2.5-flash',
    contents='Some prompt here',
)

# Check if response was blocked
if response.prompt_feedback and response.prompt_feedback.block_reason:
    print(f'Prompt blocked: {response.prompt_feedback.block_reason}')
elif not response.text:
    # Check candidate finish reason
    for candidate in response.candidates:
        if candidate.finish_reason != 'STOP':
            print(f'Response stopped: {candidate.finish_reason}')
else:
    print(response.text)
```

### Model Selection Guidelines

| Use Case | Recommended Model | Why |
|----------|------------------|-----|
| General chat / assistants | `gemini-2.5-flash` | Best price-performance, stable, fast |
| Complex reasoning / coding | `gemini-2.5-pro` or `gemini-3-pro-preview` | Highest accuracy on benchmarks |
| High-volume / batch | `gemini-2.5-flash-lite` | Cheapest ($0.10/1M input) |
| Agentic workflows | `gemini-3-flash-preview` | Designed for agents, good tool use |
| Image understanding | `gemini-2.5-flash` | Multimodal, free tier available |
| Image generation | `gemini-2.5-flash` (native) or `imagen-4.0-generate-001` | Depends on quality vs cost |
| Audio / real-time | `gemini-2.5-flash-live-preview` | Live API with bidirectional audio |
| Embeddings | `gemini-embedding-001` | Purpose-built for vector search |

### Cost Optimization

1. **Use context caching** for repeated system prompts or large documents (90% savings on cached tokens)
2. **Use batch API** for non-time-sensitive workloads (50% savings)
3. **Choose the right model**: Flash-Lite is 12.5x cheaper than Pro for input tokens
4. **Stay under 200K tokens** to avoid the higher tier pricing on Pro models
5. **Use structured output** to avoid wasting tokens on formatting instructions
6. **Use thinking_budget=0** on 2.5 models when reasoning is not needed

### Production Checklist

- [ ] Use environment variables for API keys (never hardcode)
- [ ] Implement exponential backoff with jitter for retries
- [ ] Set appropriate safety settings for your use case
- [ ] Monitor token usage and costs
- [ ] Use Vertex AI for enterprise workloads (SLA, compliance, VPC)
- [ ] Enable billing to move beyond free tier rate limits
- [ ] Use context caching for repeated large contexts
- [ ] Handle blocked/empty responses gracefully
- [ ] Set `max_output_tokens` only when you need to cap response length
- [ ] Log request IDs for debugging with Google support

---

## Citations

- [Models | Gemini API | Google AI for Developers](https://ai.google.dev/gemini-api/docs/models) — Official model listing
- [Gemini Developer API Pricing](https://ai.google.dev/gemini-api/docs/pricing) — Official pricing page
- [Vertex AI Pricing](https://cloud.google.com/vertex-ai/generative-ai/pricing) — Vertex AI pricing
- [Rate Limits | Gemini API](https://ai.google.dev/gemini-api/docs/rate-limits) — Rate limit documentation
- [googleapis/python-genai GitHub](https://github.com/googleapis/python-genai) — Official Python SDK repository
- [Google Gen AI SDK Documentation](https://googleapis.github.io/python-genai/) — SDK API reference
- [python-genai codegen_instructions.md](https://github.com/googleapis/python-genai/blob/main/codegen_instructions.md) — SDK code patterns and best practices
- [Function Calling | Gemini API](https://ai.google.dev/gemini-api/docs/function-calling) — Function calling docs
- [Structured Outputs | Gemini API](https://ai.google.dev/gemini-api/docs/structured-output) — Structured output docs
- [Safety Settings | Gemini API](https://ai.google.dev/gemini-api/docs/safety-settings) — Safety configuration
- [Gemini Developer API vs Vertex AI](https://ai.google.dev/gemini-api/docs/migrate-to-cloud) — Platform comparison
- [Gemini 2.5 Thinking Model Updates (March 2025)](https://blog.google/innovation-and-ai/models-and-research/google-deepmind/gemini-model-thinking-updates-march-2025/) — Gemini 2.5 announcement
- [Google Gemini Wikipedia](https://en.wikipedia.org/wiki/Gemini_(language_model)) — Model history and timeline
- [Gemini 3 Developer Guide](https://ai.google.dev/gemini-api/docs/gemini-3) — Gemini 3 documentation
- [Gemini 3 Flash | Google Cloud](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/models/gemini/3-flash) — Gemini 3 Flash docs
- [Gemini 3.1 Pro | Google DeepMind](https://deepmind.google/models/gemini/pro/) — Gemini 3.1 Pro announcement
