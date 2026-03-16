from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ModelPredictRequest(_message.Message):
    __slots__ = ("prompt",)
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    def __init__(self, prompt: _Optional[str] = ...) -> None: ...

class ModelPredictResponse(_message.Message):
    __slots__ = ("response", "model_name")
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    response: str
    model_name: str
    def __init__(self, response: _Optional[str] = ..., model_name: _Optional[str] = ...) -> None: ...
