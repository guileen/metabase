// Documentation site enhancements
document.addEventListener('DOMContentLoaded', function() {
    // Initialize mobile menu toggle
    initMobileMenu();

    // Initialize scroll spy for navigation
    initScrollSpy();

    // Initialize copy buttons for code blocks
    initCopyButtons();

    // Initialize search functionality
    initSearch();

    // Initialize enhanced navigation sections
    initNavigationSections();

    // Initialize page navigation
    initPageNavigation();
});

function initMobileMenu() {
    // Create mobile menu button if it doesn't exist
    if (!document.querySelector('.mobile-menu-toggle')) {
        const toggle = document.createElement('button');
        toggle.className = 'mobile-menu-toggle';
        toggle.innerHTML = '☰';
        toggle.setAttribute('aria-label', 'Toggle navigation menu');

        // Add styles for mobile menu toggle
        toggle.style.cssText = `
            display: none;
            position: fixed;
            top: 1rem;
            left: 1rem;
            z-index: 200;
            background: var(--primary-color);
            color: white;
            border: none;
            border-radius: 0.5rem;
            padding: 0.5rem;
            font-size: 1.5rem;
            cursor: pointer;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        `;

        document.body.appendChild(toggle);

        // Show toggle on mobile screens
        const mediaQuery = window.matchMedia('(max-width: 768px)');
        function updateToggleVisibility() {
            toggle.style.display = mediaQuery.matches ? 'block' : 'none';
        }

        mediaQuery.addListener(updateToggleVisibility);
        updateToggleVisibility();

        // Toggle sidebar
        toggle.addEventListener('click', function() {
            const sidebar = document.querySelector('.sidebar');
            sidebar.classList.toggle('mobile-open');
        });

        // Close sidebar when clicking outside on mobile
        document.addEventListener('click', function(e) {
            if (mediaQuery.matches &&
                !sidebar.contains(e.target) &&
                !toggle.contains(e.target) &&
                sidebar.classList.contains('mobile-open')) {
                sidebar.classList.remove('mobile-open');
            }
        });
    }
}

function initScrollSpy() {
    const headings = document.querySelectorAll('.docs-content h2, .docs-content h3');
    const navLinks = document.querySelectorAll('.nav-section a');

    if (headings.length === 0 || navLinks.length === 0) return;

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const id = entry.target.id || entry.target.textContent.toLowerCase().replace(/\s+/g, '-');

                // Remove active class from all links
                navLinks.forEach(link => link.classList.remove('active'));

                // Add active class to current section link
                navLinks.forEach(link => {
                    if (link.getAttribute('href') === `#${id}` ||
                        link.textContent.trim() === entry.target.textContent.trim()) {
                        link.classList.add('active');
                    }
                });
            }
        });
    }, {
        rootMargin: '-20% 0px -70% 0px'
    });

    // Add IDs to headings for anchor links
    headings.forEach(heading => {
        if (!heading.id) {
            heading.id = heading.textContent.toLowerCase().replace(/\s+/g, '-').replace(/[^\w-]/g, '');
        }

        // Create anchor link
        const anchor = document.createElement('a');
        anchor.href = `#${heading.id}`;
        anchor.className = 'heading-link';
        anchor.innerHTML = '#';
        anchor.style.cssText = `
            opacity: 0;
            margin-left: 0.5rem;
            color: var(--primary-color);
            text-decoration: none;
            transition: opacity 0.2s ease;
        `;

        heading.style.position = 'relative';
        heading.appendChild(anchor);

        heading.addEventListener('mouseenter', () => {
            anchor.style.opacity = '1';
        });

        heading.addEventListener('mouseleave', () => {
            anchor.style.opacity = '0';
        });

        observer.observe(heading);
    });
}

function initCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre code');

    codeBlocks.forEach(block => {
        const pre = block.parentElement;

        // Create copy button
        const copyButton = document.createElement('button');
        copyButton.className = 'copy-button';
        copyButton.textContent = 'Copy';
        copyButton.setAttribute('aria-label', 'Copy code to clipboard');

        copyButton.style.cssText = `
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            background: var(--primary-color);
            color: white;
            border: none;
            border-radius: 0.25rem;
            padding: 0.25rem 0.5rem;
            font-size: 0.75rem;
            cursor: pointer;
            opacity: 0;
            transition: opacity 0.2s ease;
        `;

        // Make pre element relative positioning
        pre.style.position = 'relative';

        // Show/hide copy button on hover
        pre.addEventListener('mouseenter', () => {
            copyButton.style.opacity = '1';
        });

        pre.addEventListener('mouseleave', () => {
            copyButton.style.opacity = '0';
        });

        // Copy functionality
        copyButton.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(block.textContent);
                copyButton.textContent = 'Copied!';
                copyButton.style.background = '#10b981';

                setTimeout(() => {
                    copyButton.textContent = 'Copy';
                    copyButton.style.background = 'var(--primary-color)';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy text: ', err);
                copyButton.textContent = 'Failed';
                copyButton.style.background = '#ef4444';

                setTimeout(() => {
                    copyButton.textContent = 'Copy';
                    copyButton.style.background = 'var(--primary-color)';
                }, 2000);
            }
        });

        pre.appendChild(copyButton);
    });
}

function initSearch() {
    // Create search box if it doesn't exist
    if (!document.querySelector('.search-box')) {
        const searchBox = document.createElement('div');
        searchBox.className = 'search-box';
        searchBox.innerHTML = `
            <input type="search" placeholder="搜索文档..." class="search-input">
            <div class="search-results" style="display: none;"></div>
        `;

        searchBox.style.cssText = `
            padding: 1rem 1.5rem;
            border-bottom: 1px solid var(--border-color);
        `;

        const input = searchBox.querySelector('.search-input');
        const results = searchBox.querySelector('.search-results');

        input.style.cssText = `
            width: 100%;
            padding: 0.5rem 1rem;
            border: 1px solid var(--border-color);
            border-radius: 0.5rem;
            font-size: 0.875rem;
            background: var(--bg-primary);
            color: var(--text-primary);
        `;

        results.style.cssText = `
            position: absolute;
            top: 100%;
            left: 1.5rem;
            right: 1.5rem;
            background: var(--bg-primary);
            border: 1px solid var(--border-color);
            border-radius: 0.5rem;
            max-height: 300px;
            overflow-y: auto;
            z-index: 1000;
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        `;

        // Insert search box after sidebar header
        const sidebarHeader = document.querySelector('.sidebar-header');
        sidebarHeader.insertAdjacentElement('afterend', searchBox);

        // Search functionality
        let searchTimeout;

        input.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            const query = e.target.value.trim();

            if (query.length < 2) {
                results.style.display = 'none';
                return;
            }

            searchTimeout = setTimeout(() => {
                performSearch(query, results);
            }, 300);
        });

        // Hide results when clicking outside
        document.addEventListener('click', (e) => {
            if (!searchBox.contains(e.target)) {
                results.style.display = 'none';
            }
        });
    }
}

// Enhanced navigation with section collapse
function initNavigationSections() {
    const sections = document.querySelectorAll('.nav-section');

    sections.forEach(section => {
        const header = section.querySelector('h3');
        const list = section.querySelector('ul');

        if (!header || !list) return;

        // Add click handler to collapse/expand sections
        header.style.cursor = 'pointer';
        header.addEventListener('click', () => {
            const isCollapsed = list.style.display === 'none';
            list.style.display = isCollapsed ? 'block' : 'none';

            // Add visual indicator
            const indicator = header.querySelector('.collapse-indicator');
            if (indicator) {
                indicator.textContent = isCollapsed ? '▼' : '▶';
            } else {
                const newIndicator = document.createElement('span');
                newIndicator.className = 'collapse-indicator';
                newIndicator.textContent = '▼';
                newIndicator.style.cssText = `
                    float: right;
                    font-size: 0.75rem;
                    transition: transform 0.2s ease;
                `;
                header.appendChild(newIndicator);
            }
        });

        // Check if section has active page
        const activeLink = section.querySelector('a.active');
        if (!activeLink) {
            // Auto-collapse sections without active pages
            list.style.display = 'none';
            const indicator = document.createElement('span');
            indicator.className = 'collapse-indicator';
            indicator.textContent = '▶';
            indicator.style.cssText = `
                float: right;
                font-size: 0.75rem;
                transition: transform 0.2s ease;
            `;
            header.appendChild(indicator);
        }
    });
}

function performSearch(query, resultsContainer) {
    // Get all pages and their content
    const pages = [
        { title: '总览', url: '/docs/overview', content: 'MetaBase 后端核心 统一API 任务队列 多租户 行级安全' },
        { title: '快速开始', url: '/docs/start', content: '启动 配置 表管理 行级安全策略 NRPC' },
        { title: '配置', url: '/docs/config', content: '配置 端口 数据库 缓存 设置' },
        { title: '架构', url: '/docs/architecture', content: 'NRPC 存储引擎 控制台 异步 无状态' },
        { title: 'NRPC', url: '/docs/nrpc', content: '消息队列 RPC 任务调度 重试 延迟队列' },
        { title: '存储引擎', url: '/docs/storage', content: 'Sqlite Pebble Redis 缓存 持久化 索引' },
        { title: '安全', url: '/docs/security', content: '认证 授权 加密 安全策略' },
        { title: '多租户', url: '/docs/multitenancy', content: '租户 隔离 数据 安全' },
        { title: '行级安全', url: '/docs/rls', content: 'RLS 行级安全 策略 权限 控制' },
        { title: '控制台', url: '/docs/console', content: '监控 日志 分析 管理' },
        { title: 'API', url: '/docs/api', content: 'API 接口 文档 请求 响应' },
        { title: '部署', url: '/docs/deploy', content: '部署 生产 环境 监控 运维' }
    ];

    // Simple search implementation
    const results = pages.filter(page => {
        const searchText = (page.title + ' ' + page.content).toLowerCase();
        return searchText.includes(query.toLowerCase());
    });

    if (results.length === 0) {
        resultsContainer.innerHTML = '<div style="padding: 1rem; color: var(--text-secondary);">未找到相关结果</div>';
    } else {
        resultsContainer.innerHTML = results.map(page => `
            <a href="${page.url}" style="
                display: block;
                padding: 0.75rem 1rem;
                border-bottom: 1px solid var(--border-color);
                text-decoration: none;
                color: var(--text-primary);
                transition: background-color 0.2s ease;
            " onmouseover="this.style.backgroundColor='var(--bg-tertiary)'"
               onmouseout="this.style.backgroundColor='transparent'">
                <div style="font-weight: 500; margin-bottom: 0.25rem;">${page.title}</div>
                <div style="font-size: 0.875rem; color: var(--text-secondary);">${page.url}</div>
            </a>
        `).join('');
    }

    resultsContainer.style.display = 'block';
}

function initPageNavigation() {
    // Add keyboard navigation
    document.addEventListener('keydown', (e) => {
        if (e.altKey) {
            const prevLink = document.querySelector('.nav-prev a');
            const nextLink = document.querySelector('.nav-next a');

            if (e.key === 'ArrowLeft' && prevLink) {
                prevLink.click();
            } else if (e.key === 'ArrowRight' && nextLink) {
                nextLink.click();
            }
        }
    });

    // Smooth scroll for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
}