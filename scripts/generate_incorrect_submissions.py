import os
import zipfile

os.makedirs('incorrect_submissions', exist_ok=True)

def make_zip(name, files):
    with zipfile.ZipFile(f'incorrect_submissions/{name}.zip', 'w', zipfile.ZIP_DEFLATED) as zf:
        for fname, content in files.items():
            zf.writestr(fname, content)

make_zip('missing_endpoints', {
    'src/main.cpp': 'int main() { return 0; }', 
    'CMakeLists.txt': 'cmake_minimum_required(VERSION 3.20)\nproject(t)\nadd_executable(engine src/main.cpp)'
})

make_zip('broken_healthcheck', {
    'src/main.cpp': 'int main() { while(1); return 0; }', 
    'CMakeLists.txt': 'cmake_minimum_required(VERSION 3.20)\nproject(t)\nadd_executable(engine src/main.cpp)', 
    'app.py': 'while True: pass', 
    'Dockerfile': 'FROM python:3.12-alpine\nCOPY app.py .\nCMD ["python", "app.py"]'
})

make_zip('syntax_error', {
    'src/main.cpp': 'int main() { return 0', 
    'CMakeLists.txt': 'cmake_minimum_required(VERSION 3.20)\nproject(t)\nadd_executable(engine src/main.cpp)'
})
