# service_python/models/ollama_predictor.py
import logging

import ollama
from grpc import StatusCode

from model.v1 import model_pb2, model_pb2_grpc

logger = logging.getLogger("python-ai")


class ModelPredictor(model_pb2_grpc.ModelPredictorServicer):
    def __init__(self, ollama_host: str, model_name: str):
        self._client = ollama.Client(host=ollama_host)
        self._model_name = model_name
        logger.info(
            "Ollama client initialized: host=%s model=%s", ollama_host, model_name
        )

    def ModelPredict(self, request, context):
        logger.info("Model predict request: prompt='%.80s...'", request.prompt)
        try:
            response = self._client.generate(
                model=self._model_name,
                prompt=request.prompt,
                options={"num_predict": 512},
            )
        except ollama.ResponseError as e:
            logger.error("Ollama response error: %s", e)
            context.set_code(StatusCode.UNAVAILABLE)
            context.set_details(f"Model backend error: {e}")
            return model_pb2.ModelPredictResponse()
        except Exception as e:
            logger.error("Unexpected error calling Ollama: %s", e)
            context.set_code(StatusCode.INTERNAL)
            context.set_details("Internal model error")
            return model_pb2.ModelPredictResponse()

        logger.info(
            "Model response: tokens_in=%s tokens_out=%s",
            response.prompt_eval_count,
            response.eval_count,
        )
        return model_pb2.ModelPredictResponse(
            response=response.response,
            model_name=self._model_name,
            prompt_eval_count=response.prompt_eval_count or 0,
            eval_count=response.eval_count or 0,
            eval_duration=response.eval_duration or 0,
        )

    def ModelPredictStream(self, request, context):
        logger.info("Model predict stream request: prompt='%.80s...'", request.prompt)
        try:
            stream = self._client.generate(
                model=self._model_name,
                prompt=request.prompt,
                options={"num_predict": 512},
                stream=True,
            )
        except ollama.ResponseError as e:
            logger.error("Ollama response error: %s", e)
            context.set_code(StatusCode.UNAVAILABLE)
            context.set_details(f"Model backend error: {e}")
            return
        except Exception as e:
            logger.error("Unexpected error calling Ollama: %s", e)
            context.set_code(StatusCode.INTERNAL)
            context.set_details("Internal model error")
            return

        try:
            for chunk in stream:
                # Only the final chunk has eval stats populated
                yield model_pb2.ModelPredictResponse(
                    response=chunk.get("response", ""),
                    model_name=self._model_name,
                    prompt_eval_count=chunk.get("prompt_eval_count") or 0,
                    eval_count=chunk.get("eval_count") or 0,
                    eval_duration=chunk.get("eval_duration") or 0,
                )
        except Exception as e:
            logger.error("Error during stream: %s", e)
            context.set_code(StatusCode.INTERNAL)
            context.set_details("Stream interrupted")
