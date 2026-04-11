# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import re

with open("docs-site/src/content/docs/release-notes.md", "r") as f:
    lines = f.readlines()

new_lines = []
in_breaking = False

for line in lines:
    if line.strip() == "### ⚠️ BREAKING CHANGES":
        new_lines.append(":::danger[BREAKING CHANGES]\n")
        in_breaking = True
    elif in_breaking and line.startswith("##"):
        new_lines.append(":::\n\n")
        new_lines.append(line)
        in_breaking = False
    else:
        new_lines.append(line)

# If the file ended while still in a breaking block
if in_breaking:
    new_lines.append(":::\n")

with open("docs-site/src/content/docs/release-notes.md", "w") as f:
    f.writelines(new_lines)
