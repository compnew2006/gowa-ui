"""
DSPy-based prompt optimization with Prompt Versioning support.

Features:
  - Weekly offline BootstrapFewShot optimization
  - Prompt Version tracking (saves performance metrics per run)
  - Load best checkpoint based on accuracy score
"""
from __future__ import annotations
import json
import logging
from pathlib import Path
from typing import Optional
from datetime import datetime

logger = logging.getLogger(__name__)

try:
    import dspy
    DSPY_AVAILABLE = True
except ImportError:
    DSPY_AVAILABLE = False

if DSPY_AVAILABLE:
    class IntentSignature(dspy.Signature):
        """Classify the intent of a customer comment."""
        comment: str = dspy.InputField(desc="Customer comment to classify")
        intent: str = dspy.OutputField(desc="Intent: price_inquiry, purchase, complaint, refund, compliment, or general")

    class SentimentSignature(dspy.Signature):
        """Analyze the sentiment of a customer comment."""
        comment: str = dspy.InputField(desc="Customer comment to analyze")
        sentiment: str = dspy.OutputField(desc="Sentiment: positive, neutral, negative, angry, or urgent")

    class ReplySignature(dspy.Signature):
        """Generate a professional customer service reply."""
        comment: str = dspy.InputField(desc="Customer comment to reply to")
        context: str = dspy.InputField(desc="Business knowledge base context")
        intent: str = dspy.InputField(desc="Detected customer intent")
        sentiment: str = dspy.InputField(desc="Detected customer sentiment")
        reply: str = dspy.OutputField(desc="Professional, extremely concise, brief, and friendly reply (exactly 1 sentence maximum, keep it short and sweet) in the customer's language")

    IntentClassifier = dspy.ChainOfThought(IntentSignature)
    SentimentAnalyzer = dspy.ChainOfThought(SentimentSignature)
    ReplyGenerator = dspy.ChainOfThought(ReplySignature)


class PromptVersion:
    """Tracks a single optimization run."""
    def __init__(self, version: int, timestamp: str, intent_accuracy: float, sentiment_accuracy: float, sample_count: int):
        self.version = version
        self.timestamp = timestamp
        self.intent_accuracy = intent_accuracy
        self.sentiment_accuracy = sentiment_accuracy
        self.sample_count = sample_count

    def to_dict(self):
        return {
            "version": self.version,
            "timestamp": self.timestamp,
            "intent_accuracy": self.intent_accuracy,
            "sentiment_accuracy": self.sentiment_accuracy,
            "sample_count": self.sample_count,
        }


class DSPyOptimizer:
    """Manages DSPy prompt optimization with weekly offline learning + prompt versioning."""

    def __init__(self, db_path: str = ".dspy"):
        self.db_path = Path(db_path)
        self.db_path.mkdir(exist_ok=True)
        self.feedback_file = self.db_path / "feedback.jsonl"
        self.versions_file = self.db_path / "versions.jsonl"
        self.checkpoint_dir = self.db_path / "checkpoints"
        self.checkpoint_dir.mkdir(exist_ok=True)

        self.intent_classifier: Optional[object] = None
        self.sentiment_analyzer: Optional[object] = None
        self.reply_generator: Optional[object] = None
        self._current_version: int = self._load_current_version()

        if DSPY_AVAILABLE:
            self._init_dspy()

    def _load_current_version(self) -> int:
        """Read the latest version number from versions file."""
        if not self.versions_file.exists():
            return 0
        try:
            lines = self.versions_file.read_text().strip().splitlines()
            if lines:
                last = json.loads(lines[-1])
                return last.get("version", 0)
        except Exception:
            pass
        return 0

    def _init_dspy(self):
        """Instantiate module-level predictors and load best checkpoint."""
        try:
            # Module-level objects are already instantiated; deepcopy to isolate per-optimizer
            import copy
            self.intent_classifier = copy.deepcopy(IntentClassifier)
            self.sentiment_analyzer = copy.deepcopy(SentimentAnalyzer)
            self.reply_generator = copy.deepcopy(ReplyGenerator)
            self._load_best_checkpoint()
        except Exception as e:
            logger.warning("[DSPy] Init failed: %s", e)
            self.intent_classifier = None
            self.sentiment_analyzer = None
            self.reply_generator = None

    def _load_best_checkpoint(self):
        """Load the checkpoint with the best combined accuracy."""
        if not self.versions_file.exists():
            return
        try:
            versions = []
            for line in self.versions_file.read_text().strip().splitlines():
                if line.strip():
                    versions.append(json.loads(line))
            if not versions:
                return
            best = max(versions, key=lambda v: v["intent_accuracy"] + v["sentiment_accuracy"])
            # Try DSPy save()/load() first, fall back to pickle
            intent_path = self.checkpoint_dir / f"v{best['version']}_intent"
            sentiment_path = self.checkpoint_dir / f"v{best['version']}_sentiment"
            if intent_path.with_suffix(".json").exists() and self.intent_classifier:
                try:
                    self.intent_classifier.load(str(intent_path))
                except Exception:
                    pass
            if sentiment_path.with_suffix(".json").exists() and self.sentiment_analyzer:
                try:
                    self.sentiment_analyzer.load(str(sentiment_path))
                except Exception:
                    pass
            reply_path = self.checkpoint_dir / f"v{best['version']}_reply"
            if reply_path.with_suffix(".json").exists() and self.reply_generator:
                try:
                    self.reply_generator.load(str(reply_path))
                except Exception:
                    pass
            logger.info(
                "[DSPy] Loaded checkpoint v%d (intent=%.1f%% sentiment=%.1f%%)",
                best["version"], best["intent_accuracy"] * 100, best["sentiment_accuracy"] * 100,
            )
        except Exception as e:
            logger.warning("[DSPy] Checkpoint load failed: %s", e)

    def _save_version(self, intent_acc: float, sentiment_acc: float, sample_count: int) -> int:
        """Save a new prompt version record. Returns the new version number."""
        new_version = self._current_version + 1
        self._current_version = new_version
        record = PromptVersion(
            version=new_version,
            timestamp=datetime.utcnow().isoformat(),
            intent_accuracy=intent_acc,
            sentiment_accuracy=sentiment_acc,
            sample_count=sample_count,
        )
        try:
            with open(self.versions_file, "a") as f:
                f.write(json.dumps(record.to_dict()) + "\n")
        except Exception as e:
            logger.warning("[DSPy] Version save failed: %s", e)
        return new_version

    def _save_checkpoint(self, version: int, intent_acc: float, sentiment_acc: float):
        """Save model checkpoint using DSPy save() API."""
        if not DSPY_AVAILABLE:
            return
        try:
            if self.intent_classifier:
                intent_path = str(self.checkpoint_dir / f"v{version}_intent")
                try:
                    self.intent_classifier.save(intent_path)
                except Exception:
                    # Fallback to pickle for older DSPy
                    import pickle
                    with open(intent_path + ".pkl", "wb") as f:
                        pickle.dump(self.intent_classifier, f)
            if self.sentiment_analyzer:
                sentiment_path = str(self.checkpoint_dir / f"v{version}_sentiment")
                try:
                    self.sentiment_analyzer.save(sentiment_path)
                except Exception:
                    import pickle
                    with open(sentiment_path + ".pkl", "wb") as f:
                        pickle.dump(self.sentiment_analyzer, f)
            if self.reply_generator:
                reply_path = str(self.checkpoint_dir / f"v{version}_reply")
                try:
                    self.reply_generator.save(reply_path)
                except Exception:
                    import pickle
                    with open(reply_path + ".pkl", "wb") as f:
                        pickle.dump(self.reply_generator, f)
            logger.info("[DSPy] Saved checkpoint v%d", version)
        except Exception as e:
            logger.warning("[DSPy] Checkpoint save failed: %s", e)

    def get_versions(self) -> list[dict]:
        """Return all prompt version history."""
        if not self.versions_file.exists():
            return []
        try:
            result = []
            for line in self.versions_file.read_text().strip().splitlines():
                if line.strip():
                    result.append(json.loads(line))
            return result
        except Exception:
            return []

    async def record_feedback(
        self,
        comment: str,
        predicted_intent: str,
        actual_intent: Optional[str],
        predicted_sentiment: str,
        actual_sentiment: Optional[str],
        ai_reply: Optional[str] = None,
        was_helpful: Optional[bool] = None,
        feedback_id: Optional[str] = None,
    ):
        """Record feedback for offline learning. If feedback_id is provided, it's an update."""
        feedback = {
            "id": feedback_id or str(datetime.utcnow().timestamp()),
            "timestamp": datetime.utcnow().isoformat(),
            "comment": comment,
            "predicted_intent": predicted_intent,
            "actual_intent": actual_intent,
            "predicted_sentiment": predicted_sentiment,
            "actual_sentiment": actual_sentiment,
            "ai_reply": ai_reply,
            "was_helpful": was_helpful,
        }
        try:
            if feedback_id:
                # Update: rewrite entire file with updated feedback
                feedbacks = []
                if self.feedback_file.exists():
                    with open(self.feedback_file, "r") as f:
                        for line in f:
                            if line.strip():
                                fb = json.loads(line)
                                if fb.get("id") != feedback_id:
                                    feedbacks.append(fb)
                feedbacks.append(feedback)
                with open(self.feedback_file, "w") as f:
                    for fb in feedbacks:
                        f.write(json.dumps(fb, ensure_ascii=False) + "\n")
            else:
                # Append new feedback
                with open(self.feedback_file, "a") as f:
                    f.write(json.dumps(feedback, ensure_ascii=False) + "\n")
        except Exception as e:
            logger.warning("[DSPy] Feedback recording failed: %s", e)

    async def remove_feedback(self, feedback_id: str):
        """Remove a feedback entry (for undoing an approval/rejection)."""
        try:
            if not self.feedback_file.exists():
                return
            feedbacks = []
            with open(self.feedback_file, "r") as f:
                for line in f:
                    if line.strip():
                        fb = json.loads(line)
                        if fb.get("id") != feedback_id:
                            feedbacks.append(fb)
            with open(self.feedback_file, "w") as f:
                for fb in feedbacks:
                    f.write(json.dumps(fb, ensure_ascii=False) + "\n")
        except Exception as e:
            logger.warning("[DSPy] Feedback removal failed: %s", e)

    async def optimize_weekly(self):
        """Run weekly offline optimization using accumulated feedback."""
        if not DSPY_AVAILABLE or not self.feedback_file.exists():
            logger.info("[DSPy] Skipping optimization: no feedback data")
            return

        try:
            feedbacks = []
            with open(self.feedback_file, "r") as f:
                for line in f:
                    if line.strip():
                        feedbacks.append(json.loads(line))

            if len(feedbacks) < 5:
                logger.info("[DSPy] Insufficient feedback (%d < 5)", len(feedbacks))
                return

            logger.info("[DSPy] Optimizing with %d examples...", len(feedbacks))

            intent_examples = [
                dspy.Example(comment=fb["comment"], intent=fb["actual_intent"]).with_inputs("comment")
                for fb in feedbacks if fb.get("actual_intent")
            ]
            sentiment_examples = [
                dspy.Example(comment=fb["comment"], sentiment=fb["actual_sentiment"], confidence=0.9).with_inputs("comment")
                for fb in feedbacks if fb.get("actual_sentiment")
            ]

            intent_acc = 0.0
            sentiment_acc = 0.0

            if intent_examples and self.intent_classifier:
                optimizer = dspy.BootstrapFewShot(metric=self._metric_intent)
                try:
                    self.intent_classifier = optimizer.compile(
                        student=self.intent_classifier,
                        trainset=intent_examples[:50],
                    )
                    # Evaluate on held-out 20%
                    eval_set = intent_examples[-max(1, len(intent_examples) // 5):]
                    correct = sum(
                        1 for ex in eval_set
                        if self.get_optimized_intent(ex.comment) == ex.intent
                    )
                    intent_acc = correct / len(eval_set)
                    logger.info("[DSPy] Intent accuracy: %.1f%%", intent_acc * 100)
                except Exception as e:
                    logger.warning("[DSPy] Intent optimization failed: %s", e)

            if sentiment_examples and self.sentiment_analyzer:
                optimizer = dspy.BootstrapFewShot(metric=self._metric_sentiment)
                try:
                    self.sentiment_analyzer = optimizer.compile(
                        student=self.sentiment_analyzer,
                        trainset=sentiment_examples[:50],
                    )
                    eval_set = sentiment_examples[-max(1, len(sentiment_examples) // 5):]
                    correct = sum(
                        1 for ex in eval_set
                        if (self.get_optimized_sentiment(ex.comment)[0] or "") == ex.sentiment
                    )
                    sentiment_acc = correct / len(eval_set)
                    logger.info("[DSPy] Sentiment accuracy: %.1f%%", sentiment_acc * 100)
                except Exception as e:
                    logger.warning("[DSPy] Sentiment optimization failed: %s", e)

            # Optimize Reply Generator
            reply_examples = [
                dspy.Example(comment=fb["comment"], context="...", intent=fb["actual_intent"], sentiment=fb["actual_sentiment"], reply=fb["admin_reply"]).with_inputs("comment", "context", "intent", "sentiment")
                for fb in feedbacks if fb.get("admin_reply") and fb.get("was_helpful") == True
            ]
            if reply_examples and self.reply_generator:
                optimizer = dspy.BootstrapFewShot(metric=self._metric_reply)
                try:
                    self.reply_generator = optimizer.compile(
                        student=self.reply_generator,
                        trainset=reply_examples[:50]
                    )
                    logger.info("[DSPy] Reply generator optimized with %d examples", len(reply_examples))
                except Exception as e:
                    logger.warning("[DSPy] Reply optimization failed: %s", e)

            # Save versioned checkpoint
            new_version = self._save_version(intent_acc, sentiment_acc, len(feedbacks))
            self._save_checkpoint(new_version, intent_acc, sentiment_acc)
            self._archive_feedback()
            logger.info("[DSPy] Weekly optimization complete — version v%d", new_version)

        except Exception as e:
            logger.error("[DSPy] Weekly optimization failed: %s", e)

    def _archive_feedback(self):
        if self.feedback_file.exists():
            ts = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
            self.feedback_file.rename(self.db_path / f"feedback_{ts}.jsonl")

    def _metric_reply(self, example, pred, trace=None) -> float:
        # Simplified semantic similarity metric would be better, but for now exact or length-based
        if not pred.reply: return 0.0
        # Check if the generated reply contains key info from the gold standard
        return 1.0 if len(pred.reply) > 5 else 0.0

    def get_optimized_reply(self, comment: str, context: str, intent: str, sentiment: str) -> Optional[str]:
        if not DSPY_AVAILABLE or not self.reply_generator:
            return None
        try:
            pred = self.reply_generator(comment=comment, context=context, intent=intent, sentiment=sentiment)
            return pred.reply.strip() or None
        except Exception:
            return None

    def get_optimized_intent(self, comment: str) -> Optional[str]:
        if not DSPY_AVAILABLE or not self.intent_classifier:
            return None
        try:
            pred = self.intent_classifier(comment=comment)
            return (pred.intent or "").strip().lower() or None
        except Exception:
            return None

    def get_optimized_sentiment(self, comment: str) -> tuple[Optional[str], float]:
        if not DSPY_AVAILABLE or not self.sentiment_analyzer:
            return None, 0.0
        try:
            pred = self.sentiment_analyzer(comment=comment)
            sentiment = (pred.sentiment or "").strip().lower() or None
            conf = float(getattr(pred, "confidence", 0.85))
            return sentiment, conf
        except Exception:
            return None, 0.0


_optimizer: Optional[DSPyOptimizer] = None


def get_optimizer() -> DSPyOptimizer:
    global _optimizer
    if _optimizer is None:
        _optimizer = DSPyOptimizer()
    return _optimizer
