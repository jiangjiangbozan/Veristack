import { Info, ShieldAlert } from 'lucide-react'

/**
 * 确认效力说明：区分「已验证邮箱的业务确认」「一次性链接访问」与「可靠电子签名」，
 * 避免客户或开发方把门户结论误当作具备法律效力的电子签名。
 */
export function ConfirmationEffects() {
  return (
    <div className="space-y-3 rounded-lg border border-info/30 bg-info/5 p-4 text-sm">
      <h3 className="flex items-center gap-2 font-medium">
        <Info className="size-4 text-info" />
        这份确认代表什么
      </h3>
      <ul className="space-y-2 text-muted-foreground">
        <li>
          <strong className="text-foreground">已验证邮箱的客户确认：</strong>
          提交前必须完成邮箱验证，系统会记录验证邮箱、验证时间与逐条结论，作为项目验收的业务证据。
        </li>
        <li>
          <strong className="text-foreground">一次性分享链接：</strong>
          链接只是访问凭证，本身不证明身份；未完成邮箱验证无法提交结论。请通过可信渠道把链接发给客户验收人。
        </li>
        <li>
          <strong className="text-foreground">不等同于电子签名：</strong>
          本平台当前不提供 CA 证书或符合《电子签名法》的可靠电子签名。若合同要求签章，仍需另行完成纸质签署或第三方电子签。
        </li>
      </ul>
      <p className="flex items-start gap-2 text-xs text-muted-foreground">
        <ShieldAlert className="mt-0.5 size-3.5 shrink-0" />
        验收结论会与验证邮箱、提交时间一并记录，供后续追溯。
      </p>
    </div>
  )
}
