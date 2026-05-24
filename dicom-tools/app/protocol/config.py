import os


class Config:
    # Which protocol backend to use. Options: "auto", "basic"
    # "auto" tries each in priority order and uses the first available.
    protocol_tool: str = os.environ.get("PROTOCOL_TOOL", "auto")

    # Default tolerance percentage for numeric parameter comparisons.
    default_tolerance: float = float(
        os.environ.get("PROTOCOL_DEFAULT_TOLERANCE", "5.0")
    )


cfg = Config()
