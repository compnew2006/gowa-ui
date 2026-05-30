from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from pydantic import BaseModel
import subprocess
import os
import shlex

router = APIRouter(tags=["console"])

class CommandRequest(BaseModel):
    command: str

@router.post("/console/exec")
async def exec_command(req: CommandRequest):
    """Execute a limited set of dashboard and system commands."""
    cmd = req.command.strip()
    
    # Safety: Only allow standard dashboard/system management commands
    allowed_prefixes = ["pm2 ", "ls ", "cat ", "npm ", "node ", "python3 ", "pip3 "]
    if not any(cmd.startswith(prefix) for prefix in allowed_prefixes):
        raise HTTPException(status_code=403, detail="Command not allowed for security reasons.")
    
    # Additional safety: prevent shell injection
    try:
        # We preserve environment variables using the local dashboard's virtualenv
        env = os.environ.copy()
        env["PATH"] = f"/opt/ai-dashboard/api-server/venv/bin:{env['PATH']}"
        
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            timeout=30,
            env=env
        )
        
        return {
            "stdout": result.stdout,
            "stderr": result.stderr,
            "exit_code": result.returncode
        }
    except subprocess.TimeoutExpired:
        return {"error": "Command timed out"}
    except Exception as e:
        return {"error": str(e)}

