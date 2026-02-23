import React, { useEffect, useState, useRef } from 'react';
import { useLocation } from 'react-router-dom';
import './TopProgressBar.css';

const TopProgressBar = () => {
    const location = useLocation();
    const [progress, setProgress] = useState(0);
    const [visible, setVisible] = useState(false);

    const activeInterval = useRef(null);
    const finishTimeout = useRef(null);

    // Efeito para interceptar cliques globalmente antes mesmo do React Router mudar a Rota.
    // Assim a barrinha já entra na tela atual (dando feedback de touch) 
    // e termina suavemente na nova tela.
    useEffect(() => {
        const handleLinkClick = (e) => {
            const target = e.target.closest('a') || e.target.closest('button[data-navigate]');
            if (!target) return;

            // Ignoram-se links externos
            const href = target.getAttribute('href');
            if (href && (href.startsWith('http') || href.startsWith('mailto') || href.startsWith('tel'))) {
                return;
            }

            // Inicia visualmente
            clearInterval(activeInterval.current);
            clearTimeout(finishTimeout.current);

            setProgress(0);
            setVisible(true);

            // Dá um pulo inicial
            setTimeout(() => setProgress(20), 10);

            // Vai avançando pra não ficar congelada
            activeInterval.current = setInterval(() => {
                setProgress(prev => {
                    const inc = Math.random() * 5 + 2;
                    return prev + inc < 85 ? prev + inc : prev;
                });
            }, 500);
        };

        // Escuta na fase de captura para pegar antes dos handlers do React Router
        document.addEventListener('click', handleLinkClick, { capture: true });

        return () => {
            document.removeEventListener('click', handleLinkClick, { capture: true });
        };
    }, []);

    // Efeito principal: a Rota oficialmente mudou (Renderizou a página nova)
    useEffect(() => {
        // Significa que a página já trocou e o cache/DOM já está sendo exibido

        clearInterval(activeInterval.current);
        clearTimeout(finishTimeout.current);

        // Puxa pro 100% instantaneamente
        setProgress(100);

        // Esconde suavemente após um tempinho
        finishTimeout.current = setTimeout(() => {
            setVisible(false);
            setTimeout(() => setProgress(0), 300); // Reseta silencioso pro próximo click
        }, 150);

        return () => {
            clearInterval(activeInterval.current);
            clearTimeout(finishTimeout.current);
        };
    }, [location.pathname]); // Só roda quando a URL muda REALMENTE mudou e o React redesenhou

    return (
        <div className={`top-progress-bar-container ${!visible ? 'hidden' : ''}`}>
            <div
                className="top-progress-bar"
                style={{ width: `${progress}%` }}
            ></div>
        </div>
    );
};

export default TopProgressBar;
