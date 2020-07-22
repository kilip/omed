<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Laravel\Core;

use Illuminate\Support\ServiceProvider;
use LaravelDoctrine\Extensions\GedmoExtensionsServiceProvider;

class CoreServiceProvider extends ServiceProvider
{
    public function boot()
    {
        $this->publishes(
            [__DIR__.'/Resources/config/omed.php' => config_path('omed/core.php')],
            'config'
        );

        $this->app->register(GedmoExtensionsServiceProvider::class);
    }

    public function register()
    {
        $this->mergeConfigFrom(
            __DIR__.'/Resources/config/omed.php',
            'omed.core'
        );
    }
}
