// Tab switching functionality
document.addEventListener('DOMContentLoaded', () => {
    const tabButtons = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const targetTab = button.dataset.tab;

            // Remove active class from all buttons and contents
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));

            // Add active class to clicked button and corresponding content
            button.classList.add('active');
            document.getElementById(targetTab).classList.add('active');
        });
    });

    // OS Tab switching for installation section
    const osTabs = document.querySelectorAll('.os-tab');
    const osContents = document.querySelectorAll('.os-content');

    osTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetOS = tab.dataset.os;

            // Remove active class from all tabs and contents
            osTabs.forEach(t => t.classList.remove('active'));
            osContents.forEach(content => content.classList.remove('active'));

            // Add active class to clicked tab and corresponding content
            tab.classList.add('active');
            document.getElementById(targetOS).classList.add('active');
        });
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

    // Add scroll effect to nav
    const nav = document.querySelector('.nav');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 50) {
            nav.style.background = 'rgba(10, 10, 15, 0.95)';
        } else {
            nav.style.background = 'rgba(10, 10, 15, 0.8)';
        }
    });

    // Terminal typing effect
    const terminalBody = document.querySelector('.terminal-body code');
    if (terminalBody) {
        const originalHTML = terminalBody.innerHTML;
        terminalBody.innerHTML = '';

        let i = 0;
        const speed = 15;

        function typeWriter() {
            if (i < originalHTML.length) {
                // Handle HTML tags
                if (originalHTML.charAt(i) === '<') {
                    const endTag = originalHTML.indexOf('>', i);
                    terminalBody.innerHTML += originalHTML.substring(i, endTag + 1);
                    i = endTag + 1;
                } else {
                    terminalBody.innerHTML += originalHTML.charAt(i);
                    i++;
                }
                setTimeout(typeWriter, speed);
            }
        }

        // Start typing after a short delay
        setTimeout(typeWriter, 500);
    }

    // Copy to clipboard for code blocks
    document.querySelectorAll('.code-block').forEach(block => {
        block.style.position = 'relative';

        const copyBtn = document.createElement('button');
        copyBtn.textContent = 'Copy';
        copyBtn.style.cssText = `
            position: absolute;
            top: 8px;
            right: 8px;
            padding: 4px 12px;
            background: rgba(124, 58, 237, 0.2);
            border: 1px solid rgba(124, 58, 237, 0.3);
            border-radius: 4px;
            color: #a855f7;
            font-size: 12px;
            cursor: pointer;
            opacity: 0;
            transition: opacity 0.2s;
        `;

        block.appendChild(copyBtn);

        block.addEventListener('mouseenter', () => {
            copyBtn.style.opacity = '1';
        });

        block.addEventListener('mouseleave', () => {
            copyBtn.style.opacity = '0';
        });

        copyBtn.addEventListener('click', async () => {
            const code = block.querySelector('code').textContent;
            await navigator.clipboard.writeText(code);
            copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                copyBtn.textContent = 'Copy';
            }, 2000);
        });
    });
});
