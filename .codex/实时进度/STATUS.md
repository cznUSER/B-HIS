# B-HIS 运行状态（H01）

## 当前总状态（2026-11-01 02:14 Asia/Shanghai）

- 本机：http://127.0.0.1:8888/login；0.3.11-draft
- REG-PATIENT-SELECT-01：**有条件完成**（H06 R5）；R1=d12b761；REG-UI-FOLD-01=a87328e 已关；REG-DOC-01=4cf1d5e 已关；仍欠 INACTIVE 样本 / gitlab 推送
- 刚闭合：UI90-04C-FIX-04=**done** 关闭 tip **64ec746**（H05 实现 **4b2e605**；H06 抽验 **64ec746** 建议通过；**明确不签 90%**；未 SCORE；未 absolute Left/04b）；UI90-04C-FIX-03=**done** tip **9443941**（H05 **39a62e0**；H06 **afad64d**）；UI90-04C-FIX-02=**done** tip **f0aa482**；UI90-04C-REVERIFY-01=**done** tip **5c829e8**；UI90-04A-FIX-01=**done** tip **be229e4**（H06 抽验 tip **2b69533**）；UI90-04A-EVID-01=**done** tip **593e7ef**；DEBT-MOCK-CHAIN-ALIGN-01=**done** tip **8a71ee4**；P1-THIN-SMOKE-01=**done** tip **42f2d09**；P1-ORDER-RECENT-01=**done** tip **42f2d09**；P1-DIAG-VERSION-DIFF-01=**done** tip **e8073b7**；P1-ORDER-COMMON-01=**done** tip **120a409**；P1-DIAG-ICD-FIND-01=**done** tip **4243b19**；P1-DIAG-COMMON-01=**done** tip **929deaf**；P1-DIAG-RESUME-01=**done** tip **5fa6b06**（**P1 三刀薄包 done**）；P1-ORDER-FIND-01=**done** tip **8865146**；P1-PRINT-STATUS-01=**done** tip **c2589d7**；P0-E-INTEGRATION-01/**Eb-H18**=**通过** tip **649cb13**（**整片 P0-E-INTEGRATION-01=done**；**P0/M1=done**）
- 条件：H06 抽验建议通过已读；看板关片；未改业务源码（关片仅进度文件）；未签 UI90 90%；未 SCORE；未 absolute Left/04b
- 上一刀：UI90-04C-FIX-04=64ec746（H05=4b2e605；H06=64ec746 建议通过；不签90%；未SCORE；未absolute Left/04b）
- 下一刀 / current：**idle**
- 说明：关片节点；UserOperationsLogPanel 条件区两行网格 + 命名列宽 token 已落盘并测量抽验通过；证据 deliverables/h05/ui90-04c-fix-04-2026-11-01.md + deliverables/h06/ui90-04c-fix-04-spotcheck-2026-11-01.md；**明确不签 90%**；04C-SCORE/04A-SCORE 后置（真宿主补活前不签 SCORE）；FIX-04b（绝对 Left）默认不做
- 共享反馈原语：useSubmitFeedback + SubmitStatusBanner **已稳**（Register→Dispense；五专项 harness 全绿；禁平行栈）
- 看板：done 76 / pending 16 / total 92（83%）；current **idle**
- origin：codex/register-slot-card-20261028；关闭 tip=**64ec746**（全 **64ec7468e94f5bb178939d4ae7319552bc6b420a**，含实现 4b2e605 与 H06 抽验）；关片前 **ahead/behind 1/0**（push 待执行；禁 force/reset）
- 已知债：gitlab 192.168.1.138:8929 不通；openapi 整文件入 d12b761；contracts README tip 漂移；大量无关 dirty（禁 reset／禁混交）；完整官方 ICD / 主数据 Dict / 强制编码 / 后端检索**另立卡**；模板密度/AUTOSAVE 未做；H05b=否；mock-chain vs D5 同键冲突债**已清理**；UI90-04A **FIX 已关但仍不签 90%**；UI90-04C **FIX-02/03/04 已关但仍不签 90%**；04A-SCORE / 04C-SCORE 后置另卡；FIX-04b（绝对 Left）默认不做
- 禁项：DIAG-AUTOSAVE；官方 ICD/Dict/强制编码/新 search API/022；看板标 90% done；夹带 dirty；force/reset；公网部署；本片未跑 sync_progress.py；**关片仅看板+STATUS，未改业务源码、未 force/reset**
