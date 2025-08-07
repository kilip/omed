<?php

namespace Omed\CMS\Controller;

use _PHPStan_f9a2208af\Nette\Neon\Exception;
use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\Routing\Attribute\Route;

class SentryController extends AbstractController
{
    #[Route('/sentry-error', name: 'sentry')]
    public function homepage(){
        throw new Exception("Unhandled");
    }
}
