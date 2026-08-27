# 喵喵学习小助手

> [!WARNING]
> 本项目目前处于一个极度危险的状态，待重构。

原一个基于 Go 的刷题工具，专为福州大学政治选择题设计，默认支持毛概和习概两门课程的选择题练习，同时支持自定义题库等功能。

## ✨ 特性

- 📚 多种答题模式（速刷、答题、错题回顾）
- 🎓 支持毛概、习概以及自定义题库
- 📊 详细的答题统计和错题管理
- 💾 本地数据持久化，支持多用户
- 🚀 单文件部署，跨平台支持
- 🌐 响应式 Web 界面
- 🐱 萌化的界面设计和交互体验

## 题库说明

- 毛概选择题：题库来源于2025上半学年康老师，包含9个章节的选择题
- 习概选择题：题库来源于2024下半学年李老师的题库，合计101道选择题，并新增来自2025下半学年杨老师的题目，合计269道选择题；章节划分为导论加17个章节（共18章节）。出题老师（仅列姓氏）包括林、王、阮、潘、杨、钱、罗、黄
- 自定义

## 主要功能

- 内置题库：支持毛概、习概等课程题库。
- 多种练习模式：速刷、答题、错题回顾。
- 错题本：按课程 / 自定义题库分别记录，不会混在一起。
- 多用户本地数据：每个用户的记录、错题、设置独立保存。
- 快捷键：支持 A-E 选项、Enter、方向键、Esc，并可在设置中自定义。
- 手机访问：电脑运行后，可在同一局域网内用手机浏览器访问练习。
- 自定义题库：支持新建/编辑课程、章节、题目、选项和答案。
- JSON 导入：可直接粘贴符合格式的 JSON 题库导入为自定义题库。
- AI 生成题库：支持 OpenAI 兼容接口，把文字材料生成选择题后保存到自定义题库。

## 快速开始

从 [Releases](https://github.com/ShaddockNH3/Meow-Politics-Helper/releases) 下载对应平台的可执行文件：

- Windows：`Meow-Politics-Helper-windows-amd64.exe`
- Windows ARM64：`Meow-Politics-Helper-windows-arm64.exe`
- Linux：`Meow-Politics-Helper-linux-amd64`
- Linux ARM64：`Meow-Politics-Helper-linux-arm64`
- macOS Intel：`Meow-Politics-Helper-darwin-amd64`
- macOS Apple Silicon：`Meow-Politics-Helper-darwin-arm64`

下载后直接运行即可。Windows 用户通常双击 `.exe`，程序启动后会自动打开浏览器页面。

## 自定义题库

在首页进入“自定义题库管理”，可以：

- 新建自定义课程和章节。
- 手动增删改题目、选项、正确答案。
- 把 JSON 题库粘贴导入。
- 使用 AI API 根据文字材料生成题目。

JSON 导入支持标准数组格式，题目字段建议包含：

```json
[
  {
    "question_number": "AI-001",
    "question_type": "单选题",
    "question_text": "题干文本",
    "options": {"A": "选项A", "B": "选项B", "C": "选项C", "D": "选项D"},
    "correct_answer": "A"
  }
]
```

## AI API 说明

AI 生成功能使用 OpenAI 兼容接口，需要在页面中填写：

- Base URL，例如 `https://api.openai.com/v1` 或兼容网关地址。
- API Path，默认 `/chat/completions`。
- API Key。
- Model。

DeepSeek 或第三方网关请使用服务商实际支持的模型名。例如某些 DeepSeek 网关只接受 `deepseek-v4-pro` 或 `deepseek-v4-flash`，传入不支持的模型名会返回 400。

API Key 只保存在本机 `user_data/<用户名>/ai_config.json`，不会上传到项目服务器。本项目是本地工具，请自行保护好电脑上的用户数据目录。

当前版本暂不开放文件上传解析。PDF、DOCX、图片识别会作为未来计划处理；现在请先把材料文字复制到输入框，或在外部工具生成 JSON 后使用“导入题库（粘贴 JSON）”。

AI 生成的题目建议人工检查。不同第三方 API 或模型网关可能存在忽略材料、返回无关内容、模型名不兼容等问题。

## 手机访问

程序运行后，在页面中打开“手机访问”即可看到局域网地址。手机和电脑需要连接到同一个网络；如果打不开，请检查防火墙或网络隔离设置。

## 本地数据

用户数据默认保存在项目运行目录下的 `user_data/`：

- 用户配置和练习记录。
- 错题本。
- 自定义题库。
- AI API 配置。

发版或分享程序时，不要把自己的 `user_data/` 一起打包。

## 开发构建

需要 Go 环境。

```bash
go test ./...
go build -o Meow-Politics-Helper.exe .
```

构建后运行生成的可执行文件即可。

## 许可证

本项目使用 MIT License，详见 [LICENSE](LICENSE)。

## 反馈

欢迎通过 GitHub Issues 反馈问题或建议：

**🎉 祝学习愉快！**
