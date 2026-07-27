'use client';

import { useToast } from './Toast';
import { IconShare, IconCopy } from './Icons';

interface ShareButtonsProps {
  title: string;
  text: string;
  url: string;
}

export default function ShareButtons({ title, text, url }: ShareButtonsProps) {
  const { showToast } = useToast();

  const shareToWeChat = () => {
    showToast('请复制链接到微信分享', 'info');
  };

  const shareToWeibo = () => {
    const weiboUrl = `http://service.weibo.com/share/share.php?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&content=${encodeURIComponent(text)}`;
    window.open(weiboUrl, '_blank', 'width=600,height=400');
  };

  const shareToQQ = () => {
    const qqUrl = `https://connect.qq.com/widget/shareqq/index.html?url=${encodeURIComponent(url)}&title=${encodeURIComponent(title)}&summary=${encodeURIComponent(text)}`;
    window.open(qqUrl, '_blank', 'width=600,height=400');
  };

  const shareToEmail = () => {
    const emailUrl = `mailto:?subject=${encodeURIComponent(title)}&body=${encodeURIComponent(text)}\n\n${encodeURIComponent(url)}`;
    window.location.href = emailUrl;
  };

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(url);
      showToast('链接已复制到剪贴板', 'success');
    } catch (err) {
      console.error('复制失败:', err);
      showToast('复制失败，请手动复制链接', 'error');
    }
  };

  const shareButtons = [
    { label: '微信', onClick: shareToWeChat },
    { label: '微博', onClick: shareToWeibo },
    { label: 'QQ', onClick: shareToQQ },
    { label: '邮件', onClick: shareToEmail },
  ];

  return (
    <div className="flex flex-wrap items-center gap-2 mt-6 pt-4 border-t border-paper">
      <IconShare size={14} className="text-ink-light/40" />
      {shareButtons.map((btn) => (
        <button
          key={btn.label}
          onClick={btn.onClick}
          className="text-xs text-jade hover:text-crimson transition-colors px-3 py-1.5 rounded-md hover:bg-crimson/5"
        >
          {btn.label}
        </button>
      ))}
      <button
        onClick={copyToClipboard}
        className="flex items-center gap-1 text-xs text-jade hover:text-crimson transition-colors px-3 py-1.5 rounded-md hover:bg-crimson/5 ml-auto"
      >
        <IconCopy size={14} />
        复制链接
      </button>
    </div>
  );
}