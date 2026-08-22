import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import type { AuthFileModelItem } from '@/features/authFiles/constants';
import { isModelExcluded } from '@/features/authFiles/constants';

export type AuthFileModelsModalProps = {
  open: boolean;
  fileName: string;
  fileType: string;
  loading: boolean;
  error: 'unsupported' | null;
  models: AuthFileModelItem[];
  excluded: Record<string, string[]>;
  onClose: () => void;
  onCopyText: (text: string) => void;
};

export function AuthFileModelsModal(props: AuthFileModelsModalProps) {
  const { t } = useTranslation();
  const { open, fileName, fileType, loading, error, models, excluded, onClose, onCopyText } = props;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('auth_files.models_title', { defaultValue: '支持的模型' }) + ` - ${fileName}`}
      footer={
        <Button variant="secondary" onClick={onClose}>
          {t('common.close')}
        </Button>
      }
    >
      {loading ? (
        <div className="text-[13px] text-muted-foreground leading-[1.55]">
          {t('auth_files.models_loading', { defaultValue: '正在加载模型列表...' })}
        </div>
      ) : error === 'unsupported' ? (
        <EmptyState
          title={t('auth_files.models_unsupported', { defaultValue: '当前版本不支持此功能' })}
          description={t('auth_files.models_unsupported_desc', {
            defaultValue: '请更新 LLMHub 到最新版本后重试'
          })}
        />
      ) : models.length === 0 ? (
        <EmptyState
          title={t('auth_files.models_empty', { defaultValue: '该凭证暂无可用模型' })}
          description={t('auth_files.models_empty_desc', {
            defaultValue: '该认证凭证可能尚未被服务器加载或没有绑定任何模型'
          })}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {models.map((model) => {
            const excludedModel = isModelExcluded(model.id, fileType, excluded);
            return (
              <div
                key={model.id}
                className={`flex items-center gap-2 py-2 px-3 border flex-wrap cursor-pointer transition-colors duration-150 ${excludedModel ? 'opacity-60 bg-secondary border-dashed hover:border-destructive' : 'bg-muted border-border hover:bg-accent hover:border-primary'}`}
                onClick={() => {
                  onCopyText(model.id);
                }}
                title={
                  excludedModel
                    ? t('auth_files.models_excluded_hint', {
                        defaultValue: '此 OAuth 模型已被禁用'
                      })
                    : t('common.copy', { defaultValue: '点击复制' })
                }
              >
                <span className={`font-['Consolas','Monaco','Courier_New',monospace] text-[13px] font-semibold break-all ${excludedModel ? 'line-through text-muted-foreground' : 'text-foreground'}`}>{model.id}</span>
                {model.display_name && model.display_name !== model.id && (
                  <span className="text-[12px] text-muted-foreground shrink-0">{model.display_name}</span>
                )}
                {model.type && <span className="text-[11px] text-muted-foreground bg-secondary py-[2px] px-2 shrink-0 ml-auto">{model.type}</span>}
                {excludedModel && (
                  <span className="text-[10px] text-destructive bg-destructive/10 py-[2px] px-[6px] border border-destructive shrink-0">
                    {t('auth_files.models_excluded_badge', { defaultValue: '已禁用' })}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      )}
    </Modal>
  );
}

