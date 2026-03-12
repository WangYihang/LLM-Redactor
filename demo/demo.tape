# demo.tape
# VHS script for recording a high-impact demo of llm-prism
Output demo.gif

# Global Styles
Set FontSize 22
Set Width 1200
Set Height 800
Set Padding 40
Set Theme "Catppuccin Frappe"
Set WindowBar Colorful
Set Margin 20

# Start with a clean slate
Sleep 1s
Type "llm-prism sync" Enter
Sleep 2s
Type "clear" Enter

# Start proxy
Type "# Start llm-prism proxy" Enter
Type "export LLM_PRISM_API_KEY=sk-demo-key-for-auth" Enter
Type "llm-prism run --port 4000 --api-url http://localhost:8080 --provider deepseek &" Enter
Sleep 3s
Type "clear" Enter

# Demonstrate the leak
Type "# Sending a prompt with a sensitive API key..." Enter
Sleep 1s

# Use single quotes for the Type command itself to allow double quotes inside
Type 'curl -s -X POST http://localhost:4000/v1/chat/completions -H "Content-Type: application/json" -d "{\"messages\": [{\"role\": \"user\", \"content\": \"My key is sk-7dd4d43022ee4cc29ace0eecb7d3363e\"}]}" | jq .'
Sleep 1s
Enter

# Show interception
Sleep 3s
Type "# BOOM! Intercepted. Let's check the detection log:" Enter
Sleep 1s
Type "tail -n 1 llm-prism-detections.jsonl | jq ." Enter
Sleep 5s

# Cleanup
Type "pkill llm-prism" Enter
Sleep 1s
Type "clear" Enter
Type "# Privacy: Protected. 🛡️" Enter
Sleep 2s
