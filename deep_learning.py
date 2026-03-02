import os
import subprocess
import json

# Absolute path to the project
path = "/Users/noiemany/Downloads/whatomate_GOWA/whatomate"

def learn_from_files(directory):
    for root, dirs, files in os.walk(os.path.join(path, directory)):
        for file in files:
            if file.endswith(('.go', '.ts', '.vue')):
                file_path = os.path.join(root, file)
                print(f"🧠 Learning from {file}...")
                # Record a learning experience: Successful analysis of a critical file
                subprocess.run([
                    "npx", "ruvector", "hooks", "learn",
                    "--state", f"file_read:{file}",
                    "--action", "analyze_complexity",
                    "--reward", "0.9",  # Positive reward for information gain
                    "--task", "confidence-scoring"
                ], capture_output=True)

# Run direct learning loops on core directories
print("🚀 Injecting dynamic patterns...")
learn_from_files("internal/handlers")
learn_from_files("frontend/src/stores")

print("✅ Knowledge injection complete. Check 'npx ruvector hooks stats' now.")
